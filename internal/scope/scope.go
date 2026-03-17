package scope

import (
	"net/url"
	"regexp"
	"strings"
)

type Scope struct {
	baseHost   string
	includeRe  []*regexp.Regexp
	excludeRe  []*regexp.Regexp
	subdomains bool
}

func New(baseURL string, include, exclude []string, subdomains bool) (*Scope, error) {
	u, err := url.Parse(baseURL)
	if err != nil { return nil, err }
	s := &Scope{baseHost: u.Hostname(), subdomains: subdomains}
	for _, re := range include {
		r, err := regexp.Compile(re)
		if err != nil { return nil, err }
		s.includeRe = append(s.includeRe, r)
	}
	for _, re := range exclude {
		r, err := regexp.Compile(re)
		if err != nil { return nil, err }
		s.excludeRe = append(s.excludeRe, r)
	}
	return s, nil
}

func (s *Scope) InScope(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil { return false }
	host := u.Hostname()
	if s.subdomains {
		if host != s.baseHost && !strings.HasSuffix(host, "."+s.baseHost) { return false }
	} else {
		if host != s.baseHost { return false }
	}
	for _, re := range s.excludeRe {
		if re.MatchString(rawURL) { return false }
	}
	if len(s.includeRe) > 0 {
		for _, re := range s.includeRe {
			if re.MatchString(rawURL) { return true }
		}
		return false
	}
	return true
}
