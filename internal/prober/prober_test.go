package prober

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestProber_Online(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>")
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]Endpoint{
		{URL: srv.URL, CheckFunc: DefaultEndpoints()[0].CheckFunc},
	})

	result := p.Check(context.Background())
	if !result.Online {
		t.Error("expected online=true")
	}
	if result.PortalURL != "" {
		t.Errorf("expected empty portal URL, got %q", result.PortalURL)
	}
}

func TestProber_PortalRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hotspot" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><form action="/auth"><input type="password" name="pass"></form></html>`)
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]Endpoint{
		{
			URL: srv.URL + "/hotspot",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return false
			},
		},
	})

	result := p.Check(context.Background())
	if result.Online {
		t.Error("expected online=false")
	}
	if !strings.HasSuffix(result.FinalURL, "/login") {
		t.Errorf("expected final URL ending with /login, got %q", result.FinalURL)
	}
}

func TestProber_204Online(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]Endpoint{
		{URL: srv.URL, CheckFunc: DefaultEndpoints()[1].CheckFunc},
	})

	result := p.Check(context.Background())
	if !result.Online {
		t.Error("expected 204 to mean online")
	}
}

func TestExtractPortalURL_MetaRefresh(t *testing.T) {
	html := `<html><head><meta http-equiv="refresh" content="0;url=https://portal.example.com/login"></head></html>`
	u := ExtractPortalURL(html, "http://captive.apple.com/hotspot-detect.html")
	if u != "https://portal.example.com/login" {
		t.Errorf("expected portal URL, got %q", u)
	}
}

func TestExtractPortalURL_FormAction(t *testing.T) {
	html := `<html><form action="/auth" method="POST"><input type="text" name="user"></form></html>`
	u := ExtractPortalURL(html, "http://portal.example.com/login")
	if u != "http://portal.example.com/auth" {
		t.Errorf("expected form action URL, got %q", u)
	}
}

func TestExtractPortalURL_JSRedirect(t *testing.T) {
	html := `<script>location.href = "https://portal.example.com/login";</script>`
	u := ExtractPortalURL(html, "http://captive.apple.com/")
	if u != "https://portal.example.com/login" {
		t.Errorf("expected JS redirect URL, got %q", u)
	}
}

func TestExtractPortalURL_Empty(t *testing.T) {
	u := ExtractPortalURL("", "http://example.com")
	if u != "" {
		t.Errorf("expected empty, got %q", u)
	}
}

func TestBackoff(t *testing.T) {
	for i := 1; i <= 5; i++ {
		d := backoff(i)
		if d <= 0 {
			t.Errorf("attempt %d: backoff should be positive, got %v", i, d)
		}
		if d > 9*time.Second {
			t.Errorf("attempt %d: backoff %v exceeds cap", i, d)
		}
	}
}
