package mutator

import (
	"fmt"
	"path/filepath"
	"strings"
)

var suffixes = []string{
	"", "/", ".php", ".html", ".json", ".xml", ".txt", ".bak",
	".old", ".backup", ".config", ".conf", ".log", ".sql",
	"_backup", "_old", "_bak", "-backup", "-old",
	"/config", "/admin", "/debug", "/test", "/api",
	"/v1", "/v2", "/internal", "/private", "/secret",
}

var commonSegs = []string{
	"admin","api","internal","private","secret","debug","config",
	"backup","test","dev","v1","v2","user","users","account",
	"accounts","login","dashboard","panel","console","settings",
	"upload","uploads","files","docs","health","status","metrics",
	"env","data","db","graphql","swagger","manage","system",
}

func Mutate(path string) []string {
	seen := map[string]bool{}
	var out []string

	add := func(p string) {
		p = strings.TrimRight(p, ".")
		if p == "" || p == "/" || seen[p] { return }
		seen[p] = true
		out = append(out, p)
	}

	path = strings.TrimSpace(path)
	if path == "" || path == "/" { return nil }

	parts := strings.Split(strings.Trim(path, "/"), "/")
	dir   := filepath.Dir(path)
	base  := filepath.Base(path)
	ext   := filepath.Ext(base)
	stem  := strings.TrimSuffix(base, ext)

	for _, s := range suffixes { add(path + s) }
	if ext != "" { add(dir + "/" + stem) }
	if dir != "/" && dir != "." {
		for _, s := range commonSegs { add(dir + "/" + s) }
	}

	// API version bump
	for i, p := range parts {
		if strings.HasPrefix(p, "v") && len(p) <= 3 {
			var n int
			fmt.Sscanf(p, "v%d", &n)
			if n > 0 {
				for v := 1; v <= 5; v++ {
					if v != n {
						np := make([]string, len(parts))
						copy(np, parts)
						np[i] = fmt.Sprintf("v%d", v)
						add("/" + strings.Join(np, "/"))
					}
				}
			}
		}
	}

	// Parent paths
	cur := ""
	for _, p := range parts { cur += "/" + p; add(cur) }

	// Children
	clean := strings.TrimRight(path, "/")
	for _, s := range commonSegs { add(clean + "/" + s) }

	// Hidden
	if !strings.HasPrefix(base, ".") {
		add(dir + "/." + stem)
	}
	add(path + "~")
	add(path + ".bak")
	add(path + ".backup")
	if ext != "" { add(dir + "/" + stem + ".bak" + ext) }

	return out
}

func MutateAll(paths []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, p := range paths {
		for _, m := range Mutate(p) {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	return result
}
