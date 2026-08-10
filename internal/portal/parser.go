package portal

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ErrRedirect is returned when no form is found but a redirect target is detected
// (iframe, frame, meta-refresh, or JS location assignment).
type ErrRedirect struct {
	URL string
}

func (e *ErrRedirect) Error() string {
	return "portal: redirect to " + e.URL
}

// jsRedirectRe matches common JS location-redirect patterns.
var jsRedirectRe = regexp.MustCompile(
	`location\.(?:href|replace|assign)\s*[=(]\s*["']([^"']+)["']|window\.location\s*=\s*["']([^"']+)["']`,
)

// FormData represents a parsed captive portal login form.
type FormData struct {
	Action        string
	Method        string
	Fields        map[string]string
	UsernameField string
	PasswordField string
	FormIndex     int
	PageURL       string // URL of the page where the form was parsed from (used for Referer header)
	AjaxHint      bool   // true for JS-synthesized forms (Ruijie etc.)
}

// ErrNoForm is returned when no form element is found on the page.
var ErrNoForm = errors.New("portal: no form element found on page")

// usernameKeywords are substrings that suggest a field is a username/ID input.
var usernameKeywords = []string{
	"user", "login", "email", "phone", "mobile",
	"member", "account", "id", "telephone",
}

// ParseDoc parses raw HTML from r into a DOM tree. Callers can use the
// returned node with ExtractScriptURLs, IsRuijiePortal, etc. without
// re-fetching the page.
func ParseDoc(r io.Reader) (*html.Node, error) {
	return html.Parse(r)
}

// ParseLoginForm parses HTML from body to find the best login form.
// Uses a proper DOM parser (golang.org/x/net/html) and scores forms
// to pick the one most likely to be a login form.
func ParseLoginForm(body io.Reader, baseURL string) (*FormData, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("portal: parse HTML: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("portal: parse base URL: %w", err)
	}

	forms := findForms(doc)
	if len(forms) == 0 {
		// Check for JS/iframe/meta-refresh redirect first.
		if redir := extractRedirectFromDoc(doc, base); redir != "" {
			return nil, &ErrRedirect{URL: redir}
		}
		// Fallback: synthesise a form from id-attributed inputs.
		// JS-driven portals (e.g. Ruijie/H3C) render inputs with id= but no
		// enclosing <form> element and submit via onclick / AJAX.
		if fd := synthesizeFormFromIDs(doc, base); fd != nil {
			return fd, nil
		}
		return nil, ErrNoForm
	}

	var best *FormData
	bestScore := -1

	for i, formNode := range forms {
		fd := extractFormData(formNode, base, i)
		score := scoreForm(fd)
		if score > bestScore {
			bestScore = score
			best = fd
		}
	}

	if best == nil {
		return nil, ErrNoForm
	}

	best.PageURL = baseURL

	// If the "best" form has no username, password, or fields (e.g. search box or empty form)
	// but the page contains a redirect (meta-refresh/JS/iframe), prefer the redirect target.
	if best.UsernameField == "" && best.PasswordField == "" && len(best.Fields) == 0 {
		if redir := extractRedirectFromDoc(doc, base); redir != "" {
			return nil, &ErrRedirect{URL: redir}
		}
	}

	return best, nil
}

// findForms walks the DOM tree and collects all <form> nodes.
func findForms(n *html.Node) []*html.Node {
	var forms []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			forms = append(forms, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return forms
}

// extractRedirectFromDoc searches the parsed DOM for redirect hints when no
// <form> is present. It checks (in order):
//  1. <iframe src> / <frame src>
//  2. <meta http-equiv="refresh" content="…;url=…">
//  3. Inline <script> content for JS location assignments
//
// Returns the resolved absolute URL, or "" if nothing is found.
func extractRedirectFromDoc(doc *html.Node, base *url.URL) string {
	var result string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "iframe", "frame":
				if src := getAttr(n, "src"); src != "" {
					if resolved, err := base.Parse(src); err == nil {
						result = resolved.String()
						return
					}
				}
			case "meta":
				if strings.EqualFold(getAttr(n, "http-equiv"), "refresh") {
					content := getAttr(n, "content")
					// Format: "N;url=TARGET" or "N;URL=TARGET"
					if idx := strings.Index(strings.ToLower(content), "url="); idx >= 0 {
						target := content[idx+4:]
						if resolved, err := base.Parse(target); err == nil {
							result = resolved.String()
							return
						}
					}
				}
			case "script":
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					script := n.FirstChild.Data
					if m := jsRedirectRe.FindStringSubmatch(script); m != nil {
						// Group 1: location.href/replace/assign; group 2: window.location
						target := m[1]
						if target == "" {
							target = m[2]
						}
						if target != "" {
							if resolved, err := base.Parse(target); err == nil {
								result = resolved.String()
								return
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return result
}

// extractFormData extracts form metadata and input fields from a <form> node.
func extractFormData(formNode *html.Node, base *url.URL, index int) *FormData {
	fd := &FormData{
		Fields:    make(map[string]string),
		FormIndex: index,
		PageURL:   base.String(),
	}

	fd.Action = getAttr(formNode, "action")
	fd.Method = strings.ToUpper(getAttr(formNode, "method"))
	if fd.Method == "" {
		fd.Method = "POST"
	}

	if fd.Action == "" {
		fd.Action = base.String()
	} else {
		if resolved, err := base.Parse(fd.Action); err == nil {
			fd.Action = resolved.String()
		}
	}

	var textInputs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "input" || n.Data == "select" || n.Data == "textarea") {
			if hasAttr(n, "disabled") {
				return
			}
			name := getAttr(n, "name")
			if name != "" {
				inputType := strings.ToLower(getAttr(n, "type"))
				if inputType == "" {
					inputType = "text"
				}
				value := getAttr(n, "value")

				if inputType == "checkbox" {
					if value == "" {
						value = "on"
					}
					fd.Fields[name] = value
				} else if inputType == "radio" {
					hasChecked := hasAttr(n, "checked")
					_, exists := fd.Fields[name]
					if !exists || hasChecked {
						fd.Fields[name] = value
					}
				} else if inputType == "image" {
					fd.Fields[name+".x"] = "0"
					fd.Fields[name+".y"] = "0"
				} else {
					fd.Fields[name] = value
				}

				switch inputType {
				case "password":
					fd.PasswordField = name
				case "text", "email", "tel":
					textInputs = append(textInputs, name)
					lower := strings.ToLower(name)
					for _, kw := range usernameKeywords {
						if strings.Contains(lower, kw) {
							if fd.UsernameField == "" {
								fd.UsernameField = name
							}
							break
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(formNode)

	// Fallback: first text input that isn't the password field
	if fd.UsernameField == "" && fd.PasswordField != "" {
		for _, name := range textInputs {
			if name != fd.PasswordField {
				fd.UsernameField = name
				break
			}
		}
	}

	return fd
}

// scoreForm scores a form for likelihood of being a login form.
func scoreForm(fd *FormData) int {
	score := 0
	if fd.PasswordField != "" {
		score += 100
	}
	if fd.UsernameField != "" {
		score += 50
	}
	if fd.Method == "POST" {
		score += 10
	}
	if len(fd.Fields) > 0 {
		score += 5
	}
	return score
}

// synthesizeFormFromIDs builds a FormData from inputs that carry id= attributes
// but are not wrapped in a <form> element.  This handles JS-driven captive
// portals (Ruijie, H3C, Huawei ePortal) where submission is done via AJAX.
// Returns nil when no recognisable username/password inputs are found.
func synthesizeFormFromIDs(doc *html.Node, base *url.URL) *FormData {
	fd := &FormData{
		Fields:   make(map[string]string),
		Action:   base.String(), // caller should resolve action from auth.js
		Method:   "POST",
		PageURL:  base.String(),
		AjaxHint: true,
	}

	var textInputs []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			if hasAttr(n, "disabled") {
				return
			}
			// Prefer name attribute; fall back to id.
			key := getAttr(n, "name")
			if key == "" {
				key = getAttr(n, "id")
			}
			if key == "" {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}

			inputType := strings.ToLower(getAttr(n, "type"))
			if inputType == "" {
				inputType = "text"
			}

			switch inputType {
			case "text", "email", "tel", "password":
				fd.Fields[key] = getAttr(n, "value")
				if inputType == "password" {
					if fd.PasswordField == "" {
						fd.PasswordField = key
					}
				} else {
					textInputs = append(textInputs, key)
					lower := strings.ToLower(key)
					for _, kw := range usernameKeywords {
						if strings.Contains(lower, kw) && fd.UsernameField == "" {
							fd.UsernameField = key
							break
						}
					}
				}
			case "checkbox":
				val := getAttr(n, "value")
				if val == "" {
					val = "on"
				}
				fd.Fields[key] = val
			case "radio":
				val := getAttr(n, "value")
				hasChecked := hasAttr(n, "checked")
				_, exists := fd.Fields[key]
				if !exists || hasChecked {
					fd.Fields[key] = val
				}
			case "image":
				fd.Fields[key+".x"] = "0"
				fd.Fields[key+".y"] = "0"
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if fd.PasswordField == "" {
		return nil
	}

	// Fallback: first text-like field that isn't the password.
	if fd.UsernameField == "" && fd.PasswordField != "" {
		for _, name := range textInputs {
			if name != fd.PasswordField {
				fd.UsernameField = name
				break
			}
		}
	}

	return fd
}

// SynthesizeRuijieForm builds a FormData for Ruijie/H3C ePortal pages that
// render their login form entirely via JavaScript.  It checks IsRuijiePortal
// first; if the page doesn't carry Ruijie fingerprints, it returns nil.
//
// The returned FormData uses the canonical Ruijie field names (userId,
// password) and the InterFace.do?method=login endpoint.  The caller must
// populate the queryString field from the portal page URL.
func SynthesizeRuijieForm(doc *html.Node, baseURL string) *FormData {
	if !IsRuijiePortal(doc) {
		return nil
	}

	action := RuijieLoginURL(baseURL)
	if action == "" {
		return nil
	}

	return &FormData{
		Action:        action,
		Method:        "POST",
		UsernameField: "userId",
		PasswordField: "password",
		PageURL:       baseURL,
		AjaxHint:      true,
		Fields: map[string]string{
			"userId":          "",
			"password":        "",
			"service":         "",
			"queryString":     "",
			"operatorPwd":     "",
			"operatorUserId":  "",
			"validcode":       "",
			"passwordEncrypt": "false",
		},
	}
}

// getAttr retrieves an attribute value from an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// hasAttr checks if an attribute exists on an HTML node.
func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}
