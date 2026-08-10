package portal

import (
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ruijieActionRe matches Ruijie ePortal login endpoint patterns in JS source.
//
//	e.g. "InterFace.do?method=login"  (absolute or relative)
var ruijieActionRe = regexp.MustCompile(`(?i)["']([^"']*InterFace\.do\?method=login[^"']*)["']`)

// ruijiePortalAuthLoginRe matches the newer Ruijie /portalauth/login endpoint
// used in auth.js via $.ajax({ url: '/portalauth/login', ... }).
var ruijiePortalAuthLoginRe = regexp.MustCompile(`(?i)["'](/portalauth/login)["']`)

// ajaxURLRe matches the first URL argument of common AJAX helpers in JS source.
//
//	$.ajax/$.post/$.get/fetch/axios.post/axios.get (positional URL argument)
var ajaxURLRe = regexp.MustCompile(`(?i)(?:\$\.(?:ajax|post|get)\s*\(\s*["']|fetch\s*\(\s*["']|axios\.(?:post|get)\s*\(\s*["'])([^"'?#][^"']*)["']`)

// ajaxObjURLRe matches URL inside an $.ajax({ url: '...' }) object literal.
var ajaxObjURLRe = regexp.MustCompile(`(?i)\$\.ajax\s*\(\s*\{[^}]*url\s*:\s*["']([^"']+)["']`)

// xhrOpenRe matches standard XMLHttpRequest open calls in JS source.
// e.g. xhr.open("POST", "/login.do") or xmlhttp.open("GET", "auth.php")
var xhrOpenRe = regexp.MustCompile(`(?i)\.open\s*\(\s*["'](?:POST|GET)["']\s*,\s*["']([^"'?#][^"']*)["']`)

// ExtractScriptURLs returns resolved URLs for all <script src="..."> tags in
// the parsed HTML document.
func ExtractScriptURLs(doc *html.Node, base *url.URL) []string {
	var urls []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			if src := getAttr(n, "src"); src != "" {
				if resolved, err := base.Parse(src); err == nil {
					urls = append(urls, resolved.String())
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return urls
}

// FindLoginActionInScript inspects a JavaScript source string and returns the
// best candidate login submission URL resolved against baseURL, or "" if none
// is found.  Priority order:
//  1. Ruijie InterFace.do?method=login  (classic ePortal)
//  2. Ruijie /portalauth/login          (newer 2nd-gen ePortal)
//  3. $.ajax({ url: '...' })            (object-style AJAX)
//  4. $.post('url') / fetch('url')      (positional-style AJAX)
func FindLoginActionInScript(script, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// 1. Classic Ruijie InterFace.do
	if m := ruijieActionRe.FindStringSubmatch(script); m != nil {
		if resolved, err := base.Parse(m[1]); err == nil {
			return resolved.String()
		}
	}

	// 2. Newer Ruijie /portalauth/login (takes priority over generic AJAX)
	if m := ruijiePortalAuthLoginRe.FindStringSubmatch(script); m != nil {
		if resolved, err := base.Parse(m[1]); err == nil {
			return resolved.String()
		}
	}

	// 3. $.ajax({ url: '...' }) object syntax
	if m := ajaxObjURLRe.FindStringSubmatch(script); m != nil {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "http") {
			if resolved, err := base.Parse(candidate); err == nil {
				return resolved.String()
			}
		}
	}

	// 4. $.post('url') / fetch('url') positional syntax
	if m := ajaxURLRe.FindStringSubmatch(script); m != nil {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "http") {
			if resolved, err := base.Parse(candidate); err == nil {
				return resolved.String()
			}
		}
	}

	// 5. xhr.open('POST', 'url') / xhr.open('GET', 'url') syntax
	if m := xhrOpenRe.FindStringSubmatch(script); m != nil {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "http") {
			if resolved, err := base.Parse(candidate); err == nil {
				return resolved.String()
			}
		}
	}

	return ""
}

// IsRuijiePortal returns true when the HTML document contains Ruijie / H3C
// ePortal fingerprints (authpluginarray, authmodel attributes, or the
// /eportal/ path in script sources).
func IsRuijiePortal(doc *html.Node) bool {
	found := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode {
			if getAttr(n, "authpluginarray") != "" || getAttr(n, "modelnum") != "" {
				found = true
				return
			}
			// auth.js, socialAuth.js, or /eportal/ path hints used by Ruijie
			if n.Data == "script" {
				src := getAttr(n, "src")
				if strings.Contains(src, "auth.js") ||
					strings.Contains(src, "socialAuth.js") ||
					strings.Contains(src, "/eportal/") {
					found = true
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

// RuijieLoginURL builds the standard Ruijie ePortal login endpoint URL from
// the portal page URL.  The Ruijie ePortal always lives at the same host/port
// under /eportal/.
func RuijieLoginURL(portalURL string) string {
	u, err := url.Parse(portalURL)
	if err != nil {
		return ""
	}
	u.Path = "/eportal/InterFace.do"
	u.RawQuery = "method=login"
	return u.String()
}
