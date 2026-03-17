package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	RST = "\033[0m"
	R   = "\033[91m"
	G   = "\033[92m"
	Y   = "\033[93m"
	B   = "\033[94m"
	M   = "\033[95m"
	C   = "\033[96m"
	W   = "\033[97m"
	DIM = "\033[2m"
	BLD = "\033[1m"
)

type Finding struct {
	Path    string `json:"path"`
	URL     string `json:"url"`
	Status  int    `json:"status"`
	Size    int    `json:"size"`
	Source  string `json:"source"`
	Depth   int    `json:"depth"`
	FoundAt string `json:"found_at"`
	Redirect string `json:"redirect,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// JSONL output format (katana compatible)
type JSONLOutput struct {
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	Tag       string `json:"tag"`
	Attribute string `json:"attribute"`
	StatusCode int   `json:"status_code,omitempty"`
	Size      int    `json:"size,omitempty"`
	Depth     int    `json:"depth"`
}

type Report struct {
	Target    string    `json:"target"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	Elapsed   string    `json:"elapsed"`
	Total     int       `json:"total_found"`
	Findings  []Finding `json:"findings"`
}

var mu sync.Mutex
var jsonlFile *os.File

func StatusColor(sc int) string {
	switch {
	case sc == 200:             return fmt.Sprintf("%s%s%d%s", G, BLD, sc, RST)
	case sc == 201||sc == 204: return fmt.Sprintf("%s%d%s", G, sc, RST)
	case sc == 301||sc == 302: return fmt.Sprintf("%s%d%s", Y, sc, RST)
	case sc == 403:             return fmt.Sprintf("%s%d%s", R, sc, RST)
	case sc == 401:             return fmt.Sprintf("%s%d%s", M, sc, RST)
	case sc == 405:             return fmt.Sprintf("%s%d%s", Y, sc, RST)
	case sc == 500:             return fmt.Sprintf("%s%s%d%s", R, BLD, sc, RST)
	case sc == 404:             return fmt.Sprintf("%s%d%s", DIM, sc, RST)
	default:                    return fmt.Sprintf("%s%d%s", DIM, sc, RST)
	}
}

func SourceTag(src string) string {
	tags := map[string]string{
		"html":     fmt.Sprintf("%s[HTML]%s", C, RST),
		"js":       fmt.Sprintf("%s[JS]  %s", B, RST),
		"comment":  fmt.Sprintf("%s[CMT] %s", Y, RST),
		"mutation": fmt.Sprintf("%s[MUT] %s", M, RST),
		"robots":   fmt.Sprintf("%s[BOT] %s", G, RST),
		"sitemap":  fmt.Sprintf("%s[MAP] %s", C, RST),
		"form":     fmt.Sprintf("%s[FRM] %s", Y, RST),
		"api":      fmt.Sprintf("%s[API] %s", R, RST),
		"headless": fmt.Sprintf("%s[HDL] %s", M, RST),
		"resume":   fmt.Sprintf("%s[RES] %s", G, RST),
	}
	if t, ok := tags[src]; ok { return t }
	return fmt.Sprintf("%s[???] %s", DIM, RST)
}

func SetJSONLFile(f *os.File) { jsonlFile = f }

func PrintFinding(f Finding) {
	mu.Lock()
	defer mu.Unlock()

	if f.Status == 404 && f.Source != "comment" && f.Source != "js" && f.Source != "api" {
		return
	}

	sc  := StatusColor(f.Status)
	src := SourceTag(f.Source)
	p   := f.Path
	if len(p) > 55 { p = p[:52] + "..." }

	redir := ""
	if f.Redirect != "" {
		short := f.Redirect
		if len(short) > 30 { short = short[:30] + "..." }
		redir = fmt.Sprintf(" %s→ %s%s", Y, short, RST)
	}

	// Juicy marker
	juicyMark := ""
	juicyWords := []string{"admin","login","dashboard","panel","config","backup","secret",
		"debug","upload","shell","env","db","manage","hidden","old","bak","xmlrpc","wp-login",
		".git",".env","swagger","graphql","internal","private","install","setup","console"}
	lowerPath := strings.ToLower(f.Path)
	for _, w := range juicyWords {
		if strings.Contains(lowerPath, w) {
			juicyMark = "  " + Y + BLD + "◄ JUICY" + RST
			break
		}
	}

	fmt.Printf("  %s  %s  %s%-55s%s  %s%db%s  d:%d%s%s\n",
		sc, src, C, p, RST, DIM, f.Size, RST, f.Depth, redir, juicyMark)

	// Write JSONL
	if jsonlFile != nil {
		out := JSONLOutput{
			Timestamp:  time.Now().Format(time.RFC3339),
			URL:        f.URL,
			Source:     f.Source,
			Tag:        tagFromSource(f.Source),
			Attribute:  "",
			StatusCode: f.Status,
			Size:       f.Size,
			Depth:      f.Depth,
		}
		b, _ := json.Marshal(out)
		jsonlFile.Write(append(b, '\n'))
	}
}

func tagFromSource(src string) string {
	switch src {
	case "html":    return "href"
	case "js":      return "script"
	case "comment": return "comment"
	case "form":    return "form"
	case "api":     return "endpoint"
	default:        return src
	}
}

func PrintStats(crawled, queued, found int) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("\r  %s[~]%s Crawled:%s%d%s  Queue:%s%d%s  Found:%s%s%d%s   ",
		C, RST, G, crawled, RST, Y, queued, RST, G, BLD, found, RST)
}

func PrintSummary(report *Report) {
	line := strings.Repeat("═", 56)
	fmt.Printf("\n%s╔%s╗%s\n", M, line, RST)
	fmt.Printf("%s║%s  %s%-54s%s%s║%s\n", M, RST, BLD+W, "SCAN COMPLETE  「死神の報告」", RST, M, RST)
	fmt.Printf("%s╚%s╝%s\n\n", M, line, RST)

	byStatus := map[int]int{}
	for _, f := range report.Findings { byStatus[f.Status]++ }
	statuses := []int{}
	for s := range byStatus { statuses = append(statuses, s) }
	sort.Ints(statuses)

	fmt.Printf("  %s%-16s%s %s%s%s\n", C, "Target", RST, BLD, report.Target, RST)
	fmt.Printf("  %s%-16s%s %s\n", C, "Start", RST, report.StartTime)
	fmt.Printf("  %s%-16s%s %s\n", C, "Elapsed", RST, report.Elapsed)
	fmt.Printf("  %s%-16s%s %s%s%d%s paths\n\n", C, "Total Found", RST, G, BLD, report.Total, RST)

	fmt.Printf("  %sStatus Breakdown:%s\n", BLD+W, RST)
	labels := map[int]string{
		200:"OK",201:"Created",204:"No Content",
		301:"Moved Perm",302:"Redirect",
		400:"Bad Request",401:"Unauthorized",403:"Forbidden",
		404:"Not Found",405:"Method NA",429:"Rate Limited",
		500:"Server Error",503:"Unavailable",
	}
	for _, sc := range statuses {
		lbl := labels[sc]
		if lbl == "" { lbl = fmt.Sprintf("HTTP %d", sc) }
		fmt.Printf("    %s  %s%-16s%s %s%d paths%s\n",
			StatusColor(sc), DIM, lbl, RST, G, byStatus[sc], RST)
	}
	fmt.Println()

	bySource := map[string]int{}
	for _, f := range report.Findings { bySource[f.Source]++ }
	fmt.Printf("  %sDiscovery Sources:%s\n", BLD+W, RST)
	for _, src := range []string{"html","js","comment","api","form","mutation","robots","sitemap","headless"} {
		if n := bySource[src]; n > 0 {
			fmt.Printf("    %s  %s%d paths%s\n", SourceTag(src), G, n, RST)
		}
	}
	fmt.Println()

	fmt.Printf("  %sInteresting Finds (200/401/403):%s\n", BLD+R, RST)
	shown := 0
	for _, f := range report.Findings {
		if (f.Status == 200 || f.Status == 403 || f.Status == 401) && shown < 25 {
			fmt.Printf("    %s  %s%s%s\n", StatusColor(f.Status), C, f.Path, RST)
			shown++
		}
	}
	if shown == 0 { fmt.Printf("    %sNone%s\n", DIM, RST) }
	fmt.Println()
}

func SaveJSON(report *Report, path string) error {
	report.EndTime = time.Now().Format(time.RFC3339)
	report.Total   = len(report.Findings)
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil { return err }
	if err := os.WriteFile(path, b, 0644); err != nil { return err }
	fmt.Printf("  %s[+]%s JSON → %s%s%s\n", G, RST, BLD, path, RST)
	return nil
}

func SaveTXT(report *Report, path string) error {
	var sb strings.Builder
	for _, f := range report.Findings {
		sb.WriteString(fmt.Sprintf("%d\t%s\n", f.Status, f.URL))
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil { return err }
	fmt.Printf("  %s[+]%s TXT  → %s%s%s\n", G, RST, BLD, path, RST)
	return nil
}
