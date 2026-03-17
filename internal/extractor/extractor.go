package extractor

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Extracted struct {
	Links    []string
	Scripts  []string
	Forms    []string
	Comments []string
	APIpaths []string
}

var (
	hrefRe      = regexp.MustCompile(`(?i)(?:href|src|action|data-url|data-href|data-src)\s*=\s*["']([^"'#\s]{2,})["']`)
	fetchRe     = regexp.MustCompile(`(?i)(?:fetch|axios\.(?:get|post|put|delete|patch)|\.open\s*\(\s*["'](?:GET|POST|PUT|DELETE)["']\s*,\s*)["` + "`" + `"]([/][^"` + "`" + `'\s]{2,60})["` + "`" + `"]`)
	jspathRe    = regexp.MustCompile(`["` + "`" + `'](/[a-zA-Z0-9/_\-\.]{2,60})["` + "`" + `']`)
	commentRe   = regexp.MustCompile(`<!--([\s\S]*?)-->`)
	commentPath = regexp.MustCompile(`(/[a-zA-Z0-9/_\-\.]{2,60})`)
	apiRe       = regexp.MustCompile(`(?i)/api/v?\d*/[a-zA-Z0-9/_\-]+`)
	robotsRe    = regexp.MustCompile(`(?i)(?:Disallow|Allow):\s*(/[^\s]*)`)
	sitemapRe   = regexp.MustCompile(`<loc>(https?://[^<]+)</loc>`)
	wpRe        = regexp.MustCompile(`(?i)(?:wp-json|wp-content|wp-includes)/([a-zA-Z0-9/_\-\.]+)`)
)

var skipExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".svg": true, ".ico": true, ".woff": true, ".woff2": true,
	".ttf": true, ".eot": true, ".mp4": true, ".mp3": true,
	".pdf": true, ".zip": true, ".css": true, ".map": true,
	".webp": true, ".avif": true,
}

func Extract(body, pageURL string) *Extracted {
	e := &Extracted{}
	seen := map[string]bool{}
	base, _ := url.Parse(pageURL)

	add := func(raw string, bucket *[]string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || len(raw) < 2 { return }
		if strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "tel:") ||
			strings.HasPrefix(raw, "javascript:") || strings.HasPrefix(raw, "data:") { return }

		var path string
		switch {
		case strings.HasPrefix(raw, "http"):
			u, err := url.Parse(raw)
			if err != nil { return }
			if base != nil && u.Host != base.Host { return }
			path = u.Path
		case strings.HasPrefix(raw, "/"):
			path = raw
		default:
			if base != nil {
				rel, err := url.Parse(raw)
				if err != nil { return }
				abs := base.ResolveReference(rel)
				if abs.Host != base.Host { return }
				path = abs.Path
			}
		}

		if i := strings.Index(path, "?"); i >= 0 { path = path[:i] }
		if i := strings.Index(path, "#"); i >= 0 { path = path[:i] }
		if path == "" || path == "/" { return }

		lower := strings.ToLower(path)
		for ext := range skipExts {
			if strings.HasSuffix(lower, ext) { return }
		}

		if !seen[path] {
			seen[path] = true
			*bucket = append(*bucket, path)
		}
	}

	for _, m := range hrefRe.FindAllStringSubmatch(body, -1)    { add(m[1], &e.Links) }
	for _, m := range fetchRe.FindAllStringSubmatch(body, -1)   { add(m[1], &e.APIpaths) }
	for _, m := range jspathRe.FindAllStringSubmatch(body, -1)  { add(m[1], &e.Scripts) }
	for _, m := range apiRe.FindAllString(body, -1)             { add(m, &e.APIpaths) }
	for _, m := range wpRe.FindAllStringSubmatch(body, -1)      { add("/"+m[1], &e.APIpaths) }
	for _, m := range commentRe.FindAllStringSubmatch(body, -1) {
		for _, pm := range commentPath.FindAllStringSubmatch(m[1], -1) {
			add(pm[1], &e.Comments)
		}
	}
	return e
}

func ExtractFromRobots(body string) []string {
	var paths []string
	seen := map[string]bool{}
	for _, m := range robotsRe.FindAllStringSubmatch(body, -1) {
		p := strings.TrimSpace(m[1])
		if p != "" && p != "/" && !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	return paths
}

func ExtractFromSitemap(body, baseHost string) []string {
	var paths []string
	seen := map[string]bool{}
	for _, m := range sitemapRe.FindAllStringSubmatch(body, -1) {
		u, err := url.Parse(m[1])
		if err != nil || u.Host != baseHost { continue }
		if !seen[u.Path] {
			seen[u.Path] = true
			paths = append(paths, u.Path)
		}
	}
	return paths
}

func FetchBody(client *http.Client, rawURL string) (string, int, http.Header, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil { return "", 0, nil, err }
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) Shinigami/2.0 「死神」")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil { return "", 0, nil, err }
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(b), resp.StatusCode, resp.Header, nil
}
