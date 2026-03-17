package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"shinigami/internal/crawler"
	"shinigami/internal/waf"
	"shinigami/internal/reporter"
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

var banner = R + `
  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  ░` + M + BLD + `  ███████╗██╗  ██╗██╗███╗   ██╗██╗ ██████╗  █████╗ ███╗   ███╗██╗  ` + R + `░
  ░` + M + BLD + `  ██╔════╝██║  ██║██║████╗  ██║██║██╔════╝ ██╔══██╗████╗ ████║██║  ` + R + `░
  ░` + M + BLD + `  ███████╗███████║██║██╔██╗ ██║██║██║  ███╗███████║██╔████╔██║██║  ` + R + `░
  ░` + M + BLD + `  ╚════██║██╔══██║██║██║╚██╗██║██║██║   ██║██╔══██║██║╚██╔╝██║██║  ` + R + `░
  ░` + M + BLD + `  ███████║██║  ██║██║██║ ╚████║██║╚██████╔╝██║  ██║██║ ╚═╝ ██║██║  ` + R + `░
  ░` + M + BLD + `  ╚══════╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚═╝ ` + R + `░
  ░                                                                     ░
  ░  ` + Y + BLD + `死神 v1.0  │  Smart Spider & Directory Hunter  │  WAF Detect` + R + `  ░
  ░  ` + Y + `No wordlist needed — crawls, mutates, hunts intelligently       ` + R + `░
  ░  ` + C + `by 0xWHITEROOM  「0xホワイトルーム」                                            ` + R + `░
  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░` + RST

// multiFlag lets a flag be specified multiple times
type multiFlag []string
func (m *multiFlag) String() string         { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error     { *m = append(*m, v); return nil }

func main() {
	fmt.Println(banner)

	var (
		target    = flag.String("u", "", "Target URL  (e.g. https://example.com)")
		threads   = flag.Int("t", 20, "Concurrent threads  (default: 20)")
		maxDepth  = flag.Int("depth", 3, "Max crawl depth  (default: 3)")
		timeout   = flag.Float64("timeout", 8.0, "Request timeout seconds  (default: 8)")
		rateLimit = flag.Int("rl", 0, "Rate limit req/sec  (0 = unlimited)")
		output    = flag.String("o", "", "Save JSON output")
		outputTxt = flag.String("ot", "", "Save TXT output (status<TAB>url)")
		outputJSONL = flag.String("ojsonl", "", "Save JSONL output (katana compatible)")
		resumeFile  = flag.String("resume", "", "Resume file path (save/load scan state)")
		showAll   = flag.Bool("all", false, "Show all responses including 404")
		noMutate  = flag.Bool("no-mutate", false, "Disable path mutation engine")
		subdomains  = flag.Bool("sub", false, "Include subdomains in crawl scope")
		proxy       = flag.String("proxy", "", "HTTP/SOCKS5 proxy  (e.g. http://127.0.0.1:8080)")
		ua          = flag.String("ua", "", "Custom User-Agent")
		smartFilter = flag.Bool("smart", false, "Skip wp-content/themes/plugins/assets — show hidden dirs only")
		juicyOnly    = flag.Bool("juicy", false, "Show only juicy: login/admin/config/backup/debug etc")
		firewallOnly = flag.Bool("firewall", false, "Detect WAF/Firewall only — no crawl")
	)

	var includeFlags multiFlag
	var excludeFlags multiFlag
	var headerFlags  multiFlag
	flag.Var(&includeFlags, "include", "Include URL regex (can be used multiple times)")
	flag.Var(&excludeFlags, "exclude", "Exclude URL regex (can be used multiple times)")
	flag.Var(&headerFlags,  "H", "Custom header: Name:Value (can be used multiple times)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%sUsage:%s  shinigami -u <URL> [options]\n\n", BLD+W, RST)

		fmt.Fprintf(os.Stderr, "%sINPUT:%s\n", BLD+C, RST)
		fmt.Fprintf(os.Stderr, "   -u string          Target URL\n")
		fmt.Fprintf(os.Stderr, "   -resume string      Resume scan from file\n\n")

		fmt.Fprintf(os.Stderr, "%sCONFIGURATION:%s\n", BLD+C, RST)
		fmt.Fprintf(os.Stderr, "   -t int             Threads (default 20)\n")
		fmt.Fprintf(os.Stderr, "   -depth int         Max crawl depth (default 3)\n")
		fmt.Fprintf(os.Stderr, "   -timeout float     Request timeout sec (default 8)\n")
		fmt.Fprintf(os.Stderr, "   -rl int            Rate limit req/sec (0=unlimited)\n")
		fmt.Fprintf(os.Stderr, "   -no-mutate         Disable path mutation\n")
		fmt.Fprintf(os.Stderr, "   -proxy string      HTTP/SOCKS5 proxy URL\n")
		fmt.Fprintf(os.Stderr, "   -ua string         Custom User-Agent\n")
		fmt.Fprintf(os.Stderr, "   -H string          Custom header (Name:Value) repeatable\n\n")

		fmt.Fprintf(os.Stderr, "%sSCOPE:%s\n", BLD+C, RST)
		fmt.Fprintf(os.Stderr, "   -sub               Include subdomains\n")
		fmt.Fprintf(os.Stderr, "   -include string    In-scope regex (repeatable)\n")
		fmt.Fprintf(os.Stderr, "   -exclude string    Out-of-scope regex (repeatable)\n\n")

		fmt.Fprintf(os.Stderr, "%sOUTPUT:%s\n", BLD+C, RST)
		fmt.Fprintf(os.Stderr, "   -o string          JSON output file\n")
		fmt.Fprintf(os.Stderr, "   -ot string         TXT output file\n")
		fmt.Fprintf(os.Stderr, "   -ojsonl string     JSONL output (katana compatible)\n")
		fmt.Fprintf(os.Stderr, "   -all               Show all (including 404)\n")
		fmt.Fprintf(os.Stderr, "   -smart             Skip themes/plugins/assets/api paths\n")
		fmt.Fprintf(os.Stderr, "   -juicy             Show only juicy paths\n")
		fmt.Fprintf(os.Stderr, "   -firewall          Detect WAF/Firewall only (no crawl)\n\n")
		fmt.Fprintf(os.Stderr, "   -juicy             Show only juicy paths (admin/login/config/backup)\n\n")

		fmt.Fprintf(os.Stderr, "%sEXAMPLES:%s\n", BLD+G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -t 50 -depth 5%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -rl 10%s            (10 req/s)\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -o out.json%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -ojsonl out.jsonl%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -resume scan.json%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -sub%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -proxy http://127.0.0.1:8080%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -smart%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -smart -juicy%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com --firewall%s\n", G, RST)
		fmt.Fprintf(os.Stderr, "   %sshinigami -u https://example.com -H 'Cookie: session=abc'%s\n\n", G, RST)

		fmt.Fprintf(os.Stderr, "%sSOURCE TAGS:%s\n", BLD+Y, RST)
		fmt.Fprintf(os.Stderr, "   [HTML]  href/src/action links\n")
		fmt.Fprintf(os.Stderr, "   [JS]    JavaScript endpoints\n")
		fmt.Fprintf(os.Stderr, "   [CMT]   HTML comment hidden paths\n")
		fmt.Fprintf(os.Stderr, "   [API]   API endpoint patterns\n")
		fmt.Fprintf(os.Stderr, "   [FRM]   Form action URLs\n")
		fmt.Fprintf(os.Stderr, "   [MUT]   Smart path mutations\n")
		fmt.Fprintf(os.Stderr, "   [BOT]   robots.txt paths\n")
		fmt.Fprintf(os.Stderr, "   [MAP]   sitemap.xml paths\n\n")
	}

	flag.Parse()

	// WAF-only mode
	if *firewallOnly {
		if len(os.Args) < 2 || *target == "" {
			fmt.Fprintf(os.Stderr, "%s[-]%s -u required\n", R, RST)
			os.Exit(1)
		}
		t := *target
		if !strings.HasPrefix(t, "http") { t = "https://" + t }
		waf.Detect(t)
		os.Exit(0)
	}

	if *target == "" {
		flag.Usage()
		os.Exit(0)
	}

	if !strings.HasPrefix(*target, "http") {
		*target = "https://" + *target
	}
	*target = strings.TrimRight(*target, "/")

	// Parse headers
	headers := map[string]string{}
	for _, h := range headerFlags {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	cfg := crawler.Config{
		BaseURL:     *target,
		Threads:     *threads,
		MaxDepth:    *maxDepth,
		Timeout:     time.Duration(*timeout * float64(time.Second)),
		RateLimit:   *rateLimit,
		Output:      *output,
		OutputTXT:   *outputTxt,
		OutputJSONL: *outputJSONL,
		ResumeFile:  *resumeFile,
		ShowAll:     *showAll,
		NoMutate:    *noMutate,
		Subdomains:  *subdomains,
		Include:     []string(includeFlags),
		Exclude:     []string(excludeFlags),
		Headers:     headers,
		UserAgent:   *ua,
		Proxy:       *proxy,
		SmartFilter: *smartFilter,
		JuicyOnly:   *juicyOnly,
	}

	// Always run WAF detect first
	waf.Detect(*target)

	c, err := crawler.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s[-]%s Error: %v\n", R, RST, err)
		os.Exit(1)
	}

	report := c.Run()
	reporter.PrintSummary(report)

	if *output != "" {
		if err := reporter.SaveJSON(report, *output); err != nil {
			fmt.Fprintf(os.Stderr, "%s[-]%s Save error: %v\n", R, RST, err)
		}
	}
	if *outputTxt != "" {
		if err := reporter.SaveTXT(report, *outputTxt); err != nil {
			fmt.Fprintf(os.Stderr, "%s[-]%s Save error: %v\n", R, RST, err)
		}
	}

	fmt.Printf("\n%s%s%s\n", M, strings.Repeat("═", 58), RST)
	fmt.Printf("%s  「死神の仕事完了」  SHINIGAMI DONE  — by FIN 🌸%s\n", G+BLD, RST)
	fmt.Printf("%s%s%s\n\n", M, strings.Repeat("═", 58), RST)
}
