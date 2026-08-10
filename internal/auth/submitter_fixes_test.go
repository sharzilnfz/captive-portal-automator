package auth

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sharzilnafis/autocap/internal/credential"
	"github.com/sharzilnafis/autocap/internal/portal"
)

// ─────────────────────────────────────────────────────────────────────────────
// Headers  [POST-FIX A] — fail until Submitter sends browser-equivalent headers
// ─────────────────────────────────────────────────────────────────────────────

func TestSubmitter_SendsBrowserHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	form := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "POST",
		Fields:        map[string]string{},
		UsernameField: "user",
		PasswordField: "pass",
		PageURL:       srv.URL + "/login",
	}
	creds := &credential.Credentials{Username: "u", Password: "p"}
	if _, err := sub.Submit(context.Background(), form, creds); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	if ref := got.Get("Referer"); ref != srv.URL+"/login" {
		t.Errorf("Referer = %q, want %q", ref, srv.URL+"/login")
	}
	if org := got.Get("Origin"); org != srv.URL {
		t.Errorf("Origin = %q, want %q", org, srv.URL)
	}
	if acc := got.Get("Accept"); !strings.Contains(acc, "text/html") {
		t.Errorf("Accept = %q, want to contain text/html", acc)
	}
	if got.Get("Accept-Language") == "" {
		t.Error("Accept-Language header missing")
	}
	if ua := got.Get("User-Agent"); !strings.Contains(ua, "Mozilla") {
		t.Errorf("User-Agent = %q", ua)
	}
	if ct := got.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestSubmitter_XHRHeaderOnlyWhenAjax(t *testing.T) {
	var captured []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = append(captured, r.Header.Clone())
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	creds := &credential.Credentials{Username: "u", Password: "p"}

	mk := func(ajax bool) *portal.FormData {
		return &portal.FormData{
			Action: srv.URL + "/auth", Method: "POST",
			Fields: map[string]string{}, UsernameField: "user", PasswordField: "pass",
			PageURL: srv.URL + "/login", AjaxHint: ajax,
		}
	}
	if _, err := sub.Submit(context.Background(), mk(true), creds); err != nil {
		t.Fatalf("Submit(ajax): %v", err)
	}
	if _, err := sub.Submit(context.Background(), mk(false), creds); err != nil {
		t.Fatalf("Submit(plain): %v", err)
	}

	if got := captured[0].Get("X-Requested-With"); got != "XMLHttpRequest" {
		t.Errorf("AjaxHint=true: X-Requested-With = %q, want XMLHttpRequest", got)
	}
	if got := captured[1].Get("X-Requested-With"); got != "" {
		t.Errorf("AjaxHint=false: X-Requested-With = %q, want absent", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Session state — regression guards (pass today)
// ─────────────────────────────────────────────────────────────────────────────

func TestSubmitter_UsesClientsCookieJar(t *testing.T) {
	// Guards against Submitter silently using http.DefaultClient.
	var sawSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sess"); err == nil {
			sawSession = c.Value
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	actionURL := srv.URL + "/auth"
	u, _ := url.Parse(actionURL)
	jar.SetCookies(u, []*http.Cookie{{Name: "sess", Value: "s123", Path: "/"}})

	sub := NewSubmitter(client, testLogger())
	form := &portal.FormData{
		Action: actionURL, Method: "POST",
		Fields: map[string]string{}, UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(context.Background(), form, &credential.Credentials{Username: "u", Password: "p"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if sawSession != "s123" {
		t.Errorf("server saw cookie %q, want s123", sawSession)
	}
}

func TestSubmitter_SessionCookieSurvivesPostLoginRedirect(t *testing.T) {
	// Real-world pattern: POST /auth → Set-Cookie + 302 → /status.
	// The cookie set by the POST response must be sent on the redirect hop.
	var statusSawAuthCookie bool
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "auth", Value: "granted", Path: "/"})
		http.Redirect(w, r, "/status", http.StatusFound)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("auth"); err == nil && c.Value == "granted" {
			statusSawAuthCookie = true
		}
		w.WriteHeader(200)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	form := &portal.FormData{
		Action: srv.URL + "/auth", Method: "POST",
		Fields: map[string]string{}, UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(context.Background(), form, &credential.Credentials{Username: "u", Password: "p"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !statusSawAuthCookie {
		t.Error("cookie from POST response was not sent on the redirect follow — check CheckRedirect/jar wiring")
	}
}

func TestSubmitter_PrefersFreshFormFieldsOverStoredStaticFields(t *testing.T) {
	// Critical contract: the payload must come from the freshly parsed form.
	// creds.StaticFields (migrated from v1) may hold stale CSRF tokens and
	// must NEVER reach the wire.
	var receivedCSRF string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedCSRF = r.FormValue("csrf")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	form := &portal.FormData{
		Action: srv.URL + "/auth", Method: "POST",
		Fields:        map[string]string{"csrf": "FRESH"},
		UsernameField: "user", PasswordField: "pass",
	}
	creds := &credential.Credentials{
		Username: "u", Password: "p",
		UsernameField: "user", PasswordField: "pass",
		StaticFields:  map[string]string{"csrf": "STALE"}, // persisted garbage
	}
	if _, err := sub.Submit(context.Background(), form, creds); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if receivedCSRF != "FRESH" {
		t.Errorf("server received csrf=%q — payload must use fresh form.Fields, not stored StaticFields", receivedCSRF)
	}
}

func TestSubmitter_DoesNotMutateFormData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	form := &portal.FormData{
		Action: srv.URL + "/auth", Method: "POST",
		Fields:        map[string]string{"csrf": "tok"},
		UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(context.Background(), form, &credential.Credentials{Username: "u", Password: "p"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, injected := form.Fields["user"]; injected {
		t.Error("Submit leaked credentials into FormData.Fields")
	}
	if form.Fields["csrf"] != "tok" {
		t.Error("Submit mutated hidden fields")
	}
}

func TestSubmitter_Non2xxReturnsNoError_ContractPreserved(t *testing.T) {
	// Locked contract: Submit does not error on HTTP failure statuses;
	// the orchestrator decides via re-probing.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	sub := NewSubmitter(&http.Client{Jar: jar}, testLogger())
	form := &portal.FormData{
		Action: srv.URL + "/auth", Method: "POST",
		Fields: map[string]string{}, UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(context.Background(), form, &credential.Credentials{Username: "u", Password: "p"}); err != nil {
		t.Errorf("Submit must not error on 401, got: %v", err)
	}
}
