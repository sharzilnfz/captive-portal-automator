package portal

import (
	"errors"
	"strings"
	"testing"
)

func TestParseLoginForm_JSRedirectVariants(t *testing.T) {
	base := "http://wifi.example.com/start"
	cases := []struct {
		name string
		body string
		want string
	}{
		{"location.href double quotes",
			`<script>window.location.href = "https://portal.example.com/login";</script>`,
			"https://portal.example.com/login"},
		{"location.replace single quotes",
			`<script>location.replace('https://portal.example.com/gw');</script>`,
			"https://portal.example.com/gw"},
		{"window.location bare assignment",
			`<script>window.location = '/next';</script>`,
			"http://wifi.example.com/next"},
		{"top.location.href",
			`<script>top.location.href = '/top';</script>`,
			"http://wifi.example.com/top"},
		{"setTimeout delayed redirect",
			`<script>setTimeout(function(){ location.href = '/delayed'; }, 1500);</script>`,
			"http://wifi.example.com/delayed"},
		{"location.assign",
			`<script>location.assign("/assigned");</script>`,
			"http://wifi.example.com/assigned"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := "<html><head>" + tc.body + "</head><body></body></html>"
			_, err := ParseLoginForm(strings.NewReader(h), base)
			if err == nil {
				t.Fatal("expected ErrRedirect, got nil")
			}
			var redir *ErrRedirect
			if !errors.As(err, &redir) {
				t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
			}
			if redir.URL != tc.want {
				t.Errorf("redirect URL = %q, want %q", redir.URL, tc.want)
			}
		})
	}
}

func TestParseLoginForm_JSRedirect_DocumentLocationAssignment(t *testing.T) {
	// [POST-FIX] jsRedirectRe currently misses bare `document.location = "..."`
	// (document.location.href already works via the first alternative).
	h := `<html><head><script>document.location = "/doc-redirect";</script></head></html>`
	_, err := ParseLoginForm(strings.NewReader(h), "http://wifi.example.com/start")
	var redir *ErrRedirect
	if !errors.As(err, &redir) {
		t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
	}
	if redir.URL != "http://wifi.example.com/doc-redirect" {
		t.Errorf("redirect URL = %q", redir.URL)
	}
}

func TestParseLoginForm_MetaRefreshVariants(t *testing.T) {
	base := "http://wifi.example.com/start"
	cases := []struct {
		name string
		head string
		want string
	}{
		{"uppercase tags", `<META HTTP-EQUIV="Refresh" CONTENT="5; URL=/upper">`, "http://wifi.example.com/upper"},
		{"no space after semicolon", `<meta http-equiv="refresh" content="0;url=/nospace">`, "http://wifi.example.com/nospace"},
		{"absolute target", `<meta http-equiv="refresh" content="0; url=https://other.example.com/abs">`, "https://other.example.com/abs"},
		// Canary: some portals quote the URL inside content="".
		{"quoted url", `<meta http-equiv="refresh" content="0;url='/quoted'">`, "http://wifi.example.com/quoted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := "<html><head>" + tc.head + "</head><body></body></html>"
			_, err := ParseLoginForm(strings.NewReader(h), base)
			var redir *ErrRedirect
			if !errors.As(err, &redir) {
				t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
			}
			if redir.URL != tc.want {
				t.Errorf("redirect URL = %q, want %q", redir.URL, tc.want)
			}
		})
	}
}

func TestParseLoginForm_LoginFormBeatsJSRedirect(t *testing.T) {
	// Regression guard: when a real login form exists, JS redirect hints
	// elsewhere on the page must NOT divert parsing.
	h := `<html><head><script>window.location.href="/somewhere";</script></head>
<body><form action="/auth" method="post">
<input type="text" name="user"><input type="password" name="pass">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/login")
	if err != nil {
		t.Fatalf("expected form parse, got: %v", err)
	}
	if !strings.HasSuffix(fd.Action, "/auth") {
		t.Errorf("Action = %q", fd.Action)
	}
}
