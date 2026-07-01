package portal

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FormData represents a parsed captive portal login form.
type FormData struct {
	Action        string
	Method        string
	Fields        map[string]string
	UsernameField string
	PasswordField string
	FormIndex     int
}

// ErrNoForm is returned when no form element is found on the page.
var ErrNoForm = errors.New("portal: no form element found on page")

// usernameKeywords are substrings that suggest a field is a username/ID input.
var usernameKeywords = []string{
	"user", "login", "email", "phone", "mobile",
	"member", "account", "id", "telephone",
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

// extractFormData extracts form metadata and input fields from a <form> node.
func extractFormData(formNode *html.Node, base *url.URL, index int) *FormData {
	fd := &FormData{
		Fields:    make(map[string]string),
		FormIndex: index,
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
			name := getAttr(n, "name")
			if name == "" {
				goto next
			}

			inputType := strings.ToLower(getAttr(n, "type"))
			if inputType == "" {
				inputType = "text"
			}
			value := getAttr(n, "value")
			fd.Fields[name] = value

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
	next:
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

// getAttr retrieves an attribute value from an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
