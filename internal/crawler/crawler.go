package crawler

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"shinigami/internal/extractor"
	"shinigami/internal/mutator"
	"shinigami/internal/ratelimit"
	"shinigami/internal/reporter"
	"shinigami/internal/scope"
)

const (
	RST = "\033[0m"
	R   = "\033[91m"
	G   = "\033[92m"
	Y   = "\033[93m"
	M   = "\033[95m"
	C   = "\033[96m"
	W   = "\033[97m"
	DIM = "\033[2m"
	BLD = "\033[1m"
)

type Config struct {
	BaseURL     string
	Threads     int
	MaxDepth    int
	Timeout     time.Duration
	RateLimit   int     // requests per second, 0 = unlimited
	Output      string
	OutputTXT   string
	OutputJSONL string
	ResumeFile  string
	ShowAll     bool
	NoMutate    bool
	Subdomains  bool
	Include     []string
	Exclude     []string
	Headers      map[string]string
	UserAgent    string
	Proxy        string
	SmartFilter  bool // skip wp-content/themes/plugins/assets/api
	JuicyOnly    bool // show only 200/401/403 interesting paths
}

// noisy path prefixes to skip in SmartFilter mode
var noisyPrefixes = []string{
	"/wp-content/themes/",
	"/wp-content/plugins/",
	"/wp-content/uploads/",
	"/wp-includes/",
	"/wp-json/",
	"/wp/v2/",
	"/assets/",
	"/static/",
	"/css/",
	"/js/",
	"/fonts/",
	"/images/",
	"/img/",
	"/node_modules/",
	"/vendor/",
	"/bower_components/",
}

// noisy extensions to skip
var noisyExts = []string{
	".js", ".css", ".map", ".min.js", ".min.css",
	".woff", ".woff2", ".ttf", ".eot",
	".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp",
}

// isNoisy returns true if path should be skipped in SmartFilter mode
func isNoisy(path string) bool {
	lower := strings.ToLower(path)
	for _, p := range noisyPrefixes {
		if strings.HasPrefix(lower, p) { return true }
	}
	for _, e := range noisyExts {
		if strings.HasSuffix(lower, e) { return true }
	}
	return false
}

// isJuicy returns true if this is an interesting finding
func isJuicy(path string, status int) bool {
	// Always interesting statuses
	if status == 401 || status == 403 { return true }

	juicyWords := []string{
		"admin","login","dashboard","panel","console","config",
		"backup","secret","private","internal","debug","test",
		"api","upload","shell","cmd","env","db","database",
		"manage","control","hidden","old","bak","tmp","temp",
		"install","setup","phpmyadmin","cpanel","webmail",
		"xmlrpc","wp-login",".git",".env","swagger","graphql",
	}
	lower := strings.ToLower(path)
	for _, w := range juicyWords {
		if strings.Contains(lower, w) { return true }
	}
	return false
}

type resumeState struct {
	Visited []string `json:"visited"`
}

type job struct {
	path  string
	depth int
	src   string
}

type Crawler struct {
	cfg      Config
	client   *http.Client
	base     *url.URL
	visited  sync.Map
	queue    chan job
	findings []*reporter.Finding
	fMu      sync.Mutex
	report   *reporter.Report
	sc       *scope.Scope
	rl       *ratelimit.Limiter
	crawled  int64
	queued   int64
	found    int64
}

func New(cfg Config) (*Crawler, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil { return nil, fmt.Errorf("invalid URL: %w", err) }
	if base.Scheme == "" { base.Scheme = "https" }

	sc, err := scope.New(cfg.BaseURL, cfg.Include, cfg.Exclude, cfg.Subdomains)
	if err != nil { return nil, fmt.Errorf("scope error: %w", err) }

	// Build transport
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: cfg.Threads,
		DisableKeepAlives:   false,
	}

	// Proxy support
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	rl := ratelimit.New(cfg.RateLimit)
	_ = cfg.SmartFilter // used in process()

	return &Crawler{
		cfg:    cfg,
		client: client,
		base:   base,
		queue:  make(chan job, 20000),
		report: &reporter.Report{
			Target:    cfg.BaseURL,
			StartTime: time.Now().Format(time.RFC3339),
		},
		sc: sc,
		rl: rl,
	}, nil
}

func (c *Crawler) Run() *reporter.Report {
	printSection("SHINIGAMI SPIDER  「死神スパイダー」")

	fmt.Printf("  %s[*]%s Target     : %s%s%s\n", C, RST, BLD, c.cfg.BaseURL, RST)
	fmt.Printf("  %s[*]%s Threads    : %d\n", C, RST, c.cfg.Threads)
	fmt.Printf("  %s[*]%s Max Depth  : %d\n", C, RST, c.cfg.MaxDepth)
	fmt.Printf("  %s[*]%s Rate Limit : %d req/s\n", C, RST, c.cfg.RateLimit)
	fmt.Printf("  %s[*]%s Mutation   : %v\n", C, RST, !c.cfg.NoMutate)
	fmt.Printf("  %s[*]%s Subdomains : %v\n\n", C, RST, c.cfg.Subdomains)

	// Setup JSONL output
	if c.cfg.OutputJSONL != "" {
		f, err := os.Create(c.cfg.OutputJSONL)
		if err == nil {
			reporter.SetJSONLFile(f)
			defer f.Close()
		}
	}

	// Load resume state
	if c.cfg.ResumeFile != "" {
		c.loadResume()
	}

	fmt.Printf("  %s%-8s%-8s%-8s%-55s%-8s%s\n",
		DIM, "STATUS", "SOURCE", "DEPTH", "PATH", "SIZE", RST)
	fmt.Printf("  %s%s%s\n\n", DIM, strings.Repeat("─", 90), RST)

	start := time.Now()

	// Seed
	c.fetchSpecial("/robots.txt", "robots")
	c.fetchSpecial("/sitemap.xml", "sitemap")
	c.fetchSpecial("/.well-known/security.txt", "html")
	c.enqueue(job{path: "/", depth: 0, src: "html"})

	// Workers
	var wg sync.WaitGroup
	for i := 0; i < c.cfg.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.worker()
		}()
	}

	// Stats printer
	stopStats := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				reporter.PrintStats(
					int(atomic.LoadInt64(&c.crawled)),
					int(atomic.LoadInt64(&c.queued)),
					int(atomic.LoadInt64(&c.found)),
				)
			case <-stopStats:
				return
			}
		}
	}()

	wg.Wait()
	close(stopStats)
	fmt.Printf("\r%s\r\n", strings.Repeat(" ", 70))

	// Save resume state
	if c.cfg.ResumeFile != "" {
		c.saveResume()
	}

	elapsed := time.Since(start)
	c.report.Elapsed = elapsed.Round(time.Millisecond).String()
	c.report.Total   = len(c.findings)
	c.fMu.Lock()
	for _, f := range c.findings {
		c.report.Findings = append(c.report.Findings, *f)
	}
	c.fMu.Unlock()

	c.rl.Stop()
	return c.report
}

func (c *Crawler) worker() {
	for j := range c.queue {
		c.rl.Wait()
		c.process(j)
		atomic.AddInt64(&c.crawled, 1)
		atomic.AddInt64(&c.queued, -1)
	}
}

func (c *Crawler) process(j job) {
	if j.depth > c.cfg.MaxDepth { return }

	// SmartFilter — skip noisy paths
	if c.cfg.SmartFilter && isNoisy(j.path) { return }

	fullURL := c.base.Scheme + "://" + c.base.Host + j.path

	if !c.sc.InScope(fullURL) { return }

	body, sc, headers, err := extractor.FetchBody(c.client, fullURL)
	if err != nil { return }

	// Get redirect location
	redir := ""
	if sc == 301 || sc == 302 {
		redir = headers.Get("Location")
		// Enqueue redirect target if in scope
		if redir != "" {
			ru, err := url.Parse(redir)
			if err == nil {
				if ru.Host == "" || ru.Host == c.base.Host {
					c.enqueue(job{path: ru.Path, depth: j.depth + 1, src: j.src})
				}
			}
		}
	}

	ct := headers.Get("Content-Type")

	finding := &reporter.Finding{
		Path:        j.path,
		URL:         fullURL,
		Status:      sc,
		Size:        len(body),
		Source:      j.src,
		Depth:       j.depth,
		FoundAt:     time.Now().Format("15:04:05"),
		Redirect:    redir,
		ContentType: ct,
	}

	c.fMu.Lock()
	c.findings = append(c.findings, finding)
	c.fMu.Unlock()
	atomic.AddInt64(&c.found, 1)

	// JuicyOnly mode — only print interesting paths
	if c.cfg.JuicyOnly {
		if isJuicy(j.path, sc) {
			reporter.PrintFinding(*finding)
		}
	} else {
		reporter.PrintFinding(*finding)
	}

	if sc != 200 { return }
	if j.depth >= c.cfg.MaxDepth { return }

	ext := extractor.Extract(body, fullURL)

	for _, p := range ext.Links    { c.enqueue(job{p, j.depth+1, "html"}) }
	for _, p := range ext.Scripts  { c.enqueue(job{p, j.depth+1, "js"}) }
	for _, p := range ext.Comments { c.enqueue(job{p, j.depth+1, "comment"}) }
	for _, p := range ext.Forms    { c.enqueue(job{p, j.depth+1, "form"}) }
	for _, p := range ext.APIpaths { c.enqueue(job{p, j.depth+1, "api"}) }

	if !c.cfg.NoMutate {
		all := append(ext.Links, ext.Scripts...)
		all  = append(all, ext.APIpaths...)
		for _, m := range mutator.MutateAll(all) {
			c.enqueue(job{m, j.depth+1, "mutation"})
		}
	}
}

func (c *Crawler) fetchSpecial(path, src string) {
	fullURL := c.base.Scheme + "://" + c.base.Host + path
	body, sc, _, err := extractor.FetchBody(c.client, fullURL)
	if err != nil || sc == 404 { return }

	f := &reporter.Finding{
		Path:    path,
		URL:     fullURL,
		Status:  sc,
		Size:    len(body),
		Source:  src,
		Depth:   0,
		FoundAt: time.Now().Format("15:04:05"),
	}
	c.fMu.Lock()
	c.findings = append(c.findings, f)
	c.fMu.Unlock()
	atomic.AddInt64(&c.found, 1)
	reporter.PrintFinding(*f)

	if sc == 200 {
		if src == "robots" {
			for _, p := range extractor.ExtractFromRobots(body) {
				c.enqueue(job{p, 0, "robots"})
			}
		}
		if src == "sitemap" {
			for _, p := range extractor.ExtractFromSitemap(body, c.base.Host) {
				c.enqueue(job{p, 0, "sitemap"})
			}
		}
	}
}

func (c *Crawler) enqueue(j job) {
	if !strings.HasPrefix(j.path, "/") { j.path = "/" + j.path }
	if i := strings.Index(j.path, "?"); i >= 0 { j.path = j.path[:i] }
	if i := strings.Index(j.path, "#"); i >= 0 { j.path = j.path[:i] }
	if j.path == "" { return }

	if _, loaded := c.visited.LoadOrStore(j.path, true); loaded { return }

	atomic.AddInt64(&c.queued, 1)
	select {
	case c.queue <- j:
	default:
		atomic.AddInt64(&c.queued, -1)
	}
}

func (c *Crawler) saveResume() {
	var visited []string
	c.visited.Range(func(k, _ any) bool {
		visited = append(visited, k.(string))
		return true
	})
	state := resumeState{Visited: visited}
	b, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(c.cfg.ResumeFile, b, 0644)
	fmt.Printf("  %s[+]%s Resume saved → %s\n", G, RST, c.cfg.ResumeFile)
}

func (c *Crawler) loadResume() {
	b, err := os.ReadFile(c.cfg.ResumeFile)
	if err != nil { return }
	var state resumeState
	if err := json.Unmarshal(b, &state); err != nil { return }
	for _, p := range state.Visited {
		c.visited.Store(p, true)
	}
	fmt.Printf("  %s[*]%s Resume loaded: %s%d%s visited paths\n",
		C, RST, Y, len(state.Visited), RST)
}

func printSection(title string) {
	line := strings.Repeat("═", 56)
	fmt.Printf("\n%s╔%s╗%s\n", M, line, RST)
	fmt.Printf("%s║%s  %s%-54s%s%s║%s\n", M, RST, BLD+W, title, RST, M, RST)
	fmt.Printf("%s╚%s╝%s\n\n", M, line, RST)
}
