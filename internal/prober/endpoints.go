package prober

import (
	"regexp"
	"strings"
)

// Endpoint defines a connectivity probe target.
type Endpoint struct {
	URL       string
	CheckFunc func(status int, body string, redirected bool) bool
}

// DefaultEndpoints returns the standard probe endpoints.
func DefaultEndpoints() []Endpoint {
	return []Endpoint{
		{
			URL: "http://captive.apple.com/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && regexp.MustCompile(`(?i)<body>\s*success\s*</body>`).MatchString(body)
			},
		},
		{
			URL: "http://connectivitycheck.gstatic.com/generate_204",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return status == 204
			},
		},
		{
			URL: "http://www.msftconnecttest.com/connecttest.txt",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.TrimSpace(body) == "Microsoft Connect Test"
			},
		},
		{
			URL: "http://detectportal.firefox.com/canonical.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "success")
			},
		},
		{
			URL: "http://neverssl.com/",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && status == 200 && strings.Contains(body, "NeverSSL")
			},
		},
	}
}
