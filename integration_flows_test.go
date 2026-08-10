package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sharzilnafis/autocap/internal/auth"
	"github.com/sharzilnafis/autocap/internal/credential"
	"github.com/sharzilnafis/autocap/internal/portal"
	"github.com/sharzilnafis/autocap/internal/prober"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers — these mirror the orchestrator patterns main.go should implement
// ─────────────────────────────────────────────────────────────────────────────

func appleCheck(status int, body string, redirected bool) bool {
	return !redirected && strings.Contains(body, "Success")
}

// waitForOnline mirrors the proposed retry-based verification: portals take a
// few seconds (RADIUS/CoA) to open traffic after a successful login.
func waitForOnline(ctx context.Context, p *prober.Prober, tries int, interval time.Duration) bool {
	for i := 0; i < tries; i++ {
		if p.Check(ctx).Online {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
	return false
}

// runLoginLoop mirrors the proposed fetch→parse→submit loop: it follows
// ErrRedirect hops and keeps submitting while forms exist (multi-step portals).
func runLoginLoop(ctx context.Context, t *testing.T, client *http.Client, sub *auth.Submitter,
	startURL string, creds *credential.Credentials, maxSteps int) {
	t.Helper()
	current := startURL
	for step := 0; step < maxSteps; step++ {
		resp, err := client.Get(current)
		if err != nil {
			t.Fatalf("login loop: GET %q: %v", current, err)
		}
		form, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
		resp.Body.Close()
		if err != nil {
			var redir *portal.ErrRedirect
			switch {
			case errors.As(err, &redir):
				current = redir.URL
				continue
			case errors.Is(err, portal.ErrNoForm):
				return
			default:
				t.Fatalf("login loop: parse %q: %v", current, err)
			}
		}
		if _, err := sub.Submit(ctx, form, creds); err != nil {
			t.Fatalf("login loop: submit step %d: %v", step, err)
		}
		current = form.Action
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Multi-step portal (login form → click-through accept form)
// ─────────────────────────────────────────────────────────────────────────────

type multiStepPortal struct {
	online  bool
	step1OK bool
}

func (m *multiStepPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.online {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD></HTML>")
		} else {
			http.Redirect(w, r, "/step1", http.StatusFound)
		}
	case "/step1":
		w.WriteHeader(200)
		fmt.Fprint(w, `<form action="/step2" method="post">
<input type="hidden" name="step" value="1">
<input type="text" name="user">
<input type="password" name="pass">
</form>`)
	case "/step2":
		if r.Method == http.MethodPost {
			r.ParseForm()
			if r.FormValue("step") == "1" && r.FormValue("user") == "u" && r.FormValue("pass") == "p" {
				m.step1OK = true
			}
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `<form action="/final" method="post">
<input type="hidden" name="accept" value="true">
</form>`)
	case "/final":
		r.ParseForm()
		if m.step1OK && r.FormValue("accept") == "true" {
			m.online = true
		}
		w.WriteHeader(200)
		fmt.Fprint(w, "done")
	default:
		w.WriteHeader(404)
	}
}

func TestIntegration_MultiStepPortal(t *testing.T) {
	mock := &multiStepPortal{}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	result := p.Check(context.Background())
	if result.Online {
		t.Fatal("expected offline before login")
	}

	sub := auth.NewSubmitter(client, testLogger())
	creds := &credential.Credentials{
		Username: "u", Password: "p",
		UsernameField: "user", PasswordField: "pass",
	}
	runLoginLoop(context.Background(), t, client, sub, result.PortalURL, creds, 5)

	if !mock.online {
		t.Fatal("portal should be online after the multi-step loop")
	}
	if res := p.Check(context.Background()); !res.Online {
		t.Error("probe should report online after multi-step login")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ErrRedirect chain: probe → /gate (meta refresh) → /login (form)
// ─────────────────────────────────────────────────────────────────────────────

type redirectChainPortal struct{ online bool }

func (m *redirectChainPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.online {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD></HTML>")
		} else {
			http.Redirect(w, r, "/gate", http.StatusFound)
		}
	case "/gate":
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><head><meta http-equiv="refresh" content="0;url=/login"></head>
<body>redirecting</body></html>`)
	case "/login":
		w.WriteHeader(200)
		fmt.Fprint(w, `<form action="/auth" method="post">
<input type="hidden" name="csrf" value="rt-1">
<input type="text" name="user">
<input type="password" name="pass">
</form>`)
	case "/auth":
		r.ParseForm()
		if r.FormValue("user") == "u" && r.FormValue("pass") == "p" && r.FormValue("csrf") == "rt-1" {
			m.online = true
		}
		w.WriteHeader(200)
	default:
		w.WriteHeader(404)
	}
}

func TestIntegration_RedirectChainThenLogin(t *testing.T) {
	mock := &redirectChainPortal{}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	result := p.Check(context.Background())
	if result.Online {
		t.Fatal("expected offline")
	}

	sub := auth.NewSubmitter(client, testLogger())
	creds := &credential.Credentials{
		Username: "u", Password: "p",
		UsernameField: "user", PasswordField: "pass",
	}
	runLoginLoop(context.Background(), t, client, sub, result.PortalURL, creds, 5)

	if !mock.online {
		t.Fatal("login through meta-refresh redirect chain failed")
	}
	if res := p.Check(context.Background()); !res.Online {
		t.Error("probe should report online")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FortiGate-style failure: HTTP 200 with error text in body
// ─────────────────────────────────────────────────────────────────────────────

type errMsgPortal struct {
	online               bool
	attempts             int
	validUser, validPass string
}

func (m *errMsgPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.online {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD></HTML>")
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	case "/login":
		w.WriteHeader(200)
		fmt.Fprint(w, `<form action="/auth" method="post">
<input type="hidden" name="csrf" value="em-1">
<input type="text" name="user">
<input type="password" name="pass">
</form>`)
	case "/auth":
		m.attempts++
		r.ParseForm()
		if r.FormValue("user") == m.validUser && r.FormValue("pass") == m.validPass && r.FormValue("csrf") == "em-1" {
			m.online = true
			w.WriteHeader(200)
			fmt.Fprint(w, "ok")
			return
		}
		// 200 OK with error text — status codes lie on portals.
		w.WriteHeader(200)
		fmt.Fprint(w, "err_msg=authentication failed")
	default:
		w.WriteHeader(404)
	}
}

func TestIntegration_LoginFailure200WithErrorBody(t *testing.T) {
	mock := &errMsgPortal{validUser: "real", validPass: "correct"}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	ctx := context.Background()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})
	sub := auth.NewSubmitter(client, testLogger())

	fetchForm := func() *portal.FormData {
		resp, err := client.Get(srv.URL + "/login")
		if err != nil {
			t.Fatalf("fetch login: %v", err)
		}
		defer resp.Body.Close()
		fd, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return fd
	}

	// Wrong credentials: Submit must not error, portal must stay offline,
	// and exactly one request must hit the server (no silent retries).
	wrong := &credential.Credentials{
		Username: "wrong", Password: "wrong",
		UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(ctx, fetchForm(), wrong); err != nil {
		t.Fatalf("Submit must not error on 200-with-error: %v", err)
	}
	if mock.attempts != 1 {
		t.Errorf("expected exactly 1 attempt, got %d", mock.attempts)
	}
	if res := p.Check(ctx); res.Online {
		t.Error("must still be offline after failed login")
	}

	// Recovery with correct credentials (fresh form parse each time).
	right := &credential.Credentials{
		Username: "real", Password: "correct",
		UsernameField: "user", PasswordField: "pass",
	}
	if _, err := sub.Submit(ctx, fetchForm(), right); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res := p.Check(ctx); !res.Online {
		t.Error("probe should report online after correct login")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GET-method login (some MikroTik configs)
// ─────────────────────────────────────────────────────────────────────────────

type getMethodPortal struct{ online bool }

func (m *getMethodPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.online {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD></HTML>")
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	case "/login":
		w.WriteHeader(200)
		fmt.Fprint(w, `<form action="/auth" method="GET">
<input type="text" name="user"><input type="password" name="pass">
</form>`)
	case "/auth":
		if r.Method != http.MethodGet {
			w.WriteHeader(405)
			return
		}
		if r.URL.Query().Get("user") == "u" && r.URL.Query().Get("pass") == "p" {
			m.online = true
		}
		w.WriteHeader(200)
	default:
		w.WriteHeader(404)
	}
}

func TestIntegration_GetMethodPortal(t *testing.T) {
	mock := &getMethodPortal{}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	ctx := context.Background()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	result := p.Check(ctx)
	if result.Online {
		t.Fatal("expected offline")
	}
	resp, err := client.Get(result.PortalURL)
	if err != nil {
		t.Fatalf("fetch portal: %v", err)
	}
	fd, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
	resp.Body.Close()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.Method != "GET" {
		t.Fatalf("Method = %q, want GET", fd.Method)
	}

	sub := auth.NewSubmitter(client, testLogger())
	creds := &credential.Credentials{
		Username: "u", Password: "p",
		UsernameField: fd.UsernameField, PasswordField: fd.PasswordField,
	}
	if _, err := sub.Submit(ctx, fd, creds); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !mock.online {
		t.Error("GET-method login did not authenticate")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Self-signed HTTPS portal (UniFi :8443 / ISE style)
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_TLSPortalRequiresTrustFallback(t *testing.T) {
	mock := &mockPortal{isOnline: false, validUser: "student123", validPass: "securepass"}
	srv := httptest.NewTLSServer(mock) // self-signed cert
	defer srv.Close()

	ctx := context.Background()

	// A vanilla client must fail on the untrusted cert — this is the failure
	// mode production code needs a fallback for.
	vanilla := &http.Client{Timeout: 5 * time.Second}
	if _, err := vanilla.Get(srv.URL + "/login"); err == nil {
		t.Fatal("expected TLS verification error with vanilla client")
	} else {
		t.Logf("vanilla client failed as expected: %v", err)
	}

	// The trusted client (production: retry with InsecureSkipVerify on x509 errors).
	client := srv.Client()
	jar, _ := cookiejar.New(nil)
	client.Jar = jar

	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	result := p.Check(ctx)
	if result.Online {
		t.Fatal("expected offline")
	}
	resp, err := client.Get(result.PortalURL)
	if err != nil {
		t.Fatalf("fetch portal over TLS: %v", err)
	}
	fd, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
	resp.Body.Close()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sub := auth.NewSubmitter(client, testLogger())
	creds := &credential.Credentials{
		Username: "student123", Password: "securepass",
		UsernameField: fd.UsernameField, PasswordField: fd.PasswordField,
	}
	if _, err := sub.Submit(ctx, fd, creds); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res := p.Check(ctx); !res.Online {
		t.Error("expected online after TLS login")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Verification retries (policy propagation takes seconds)
// ─────────────────────────────────────────────────────────────────────────────

type flappingProbe struct {
	hits       int
	flipsAfter int
}

func (f *flappingProbe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.hits++
	if f.hits >= f.flipsAfter {
		w.WriteHeader(200)
		fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD></HTML>")
		return
	}
	http.Redirect(w, r, "/portal", http.StatusFound)
}

func TestIntegration_VerifyRetry_EventuallyOnline(t *testing.T) {
	f := &flappingProbe{flipsAfter: 3}
	srv := httptest.NewServer(f)
	defer srv.Close()

	p := prober.NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	ctx := context.Background()
	if res := p.Check(ctx); res.Online {
		t.Fatal("first probe must be offline")
	}
	if !waitForOnline(ctx, p, 6, 5*time.Millisecond) {
		t.Error("expected online after retries — a single-shot verify would have failed here")
	}
}

func TestIntegration_VerifyRetry_GivesUp(t *testing.T) {
	f := &flappingProbe{flipsAfter: 999}
	srv := httptest.NewServer(f)
	defer srv.Close()

	p := prober.NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	if waitForOnline(context.Background(), p, 3, time.Millisecond) {
		t.Error("waitForOnline must return false when the portal never opens")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Canary: 200-hijack probes must yield an extracted PortalURL
// ─────────────────────────────────────────────────────────────────────────────

type hijackProbe struct{}

func (hijackProbe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Portal hijacks the probe with 200 OK + its own HTML (no redirect).
	w.WriteHeader(200)
	fmt.Fprint(w, `<html><body>
<form action="/auth" method="post">
<input type="text" name="user"><input type="password" name="pass">
</form></body></html>`)
}

func TestIntegration_ProbeHijack200_PortalURLExtracted(t *testing.T) {
	// [CANARY] if this fails, prober.Check does not run ExtractPortalURL on
	// hijacked 200 bodies — set result.PortalURL from the response body there.
	srv := httptest.NewServer(hijackProbe{})
	defer srv.Close()

	p := prober.NewProber(srv.Client(), testLogger())
	p.SetEndpoints([]prober.Endpoint{{URL: srv.URL + "/hotspot-detect.html", CheckFunc: appleCheck}})

	result := p.Check(context.Background())
	if result.Online {
		t.Fatal("hijacked 200 without Success must be offline")
	}
	if !strings.HasSuffix(result.PortalURL, "/auth") {
		t.Errorf("PortalURL = %q, want extracted form action ending in /auth", result.PortalURL)
	}
}
