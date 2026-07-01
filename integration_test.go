package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sharzilnafis/capauto/internal/auth"
	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/portal"
	"github.com/sharzilnafis/capauto/internal/prober"
)

// mockPortal simulates a captive portal for integration testing.
type mockPortal struct {
	isOnline  bool
	validUser string
	validPass string
}

func (m *mockPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.isOnline {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>")
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}

	case "/login":
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><body>
			<form action="/auth" method="POST">
				<input type="hidden" name="csrf" value="mock-csrf-token">
				<input type="hidden" name="session" value="xyz987">
				<input type="text" name="username_field">
				<input type="password" name="password_field">
				<input type="submit" value="Login">
			</form>
		</body></html>`)

	case "/auth":
		r.ParseForm()
		user := r.FormValue("username_field")
		pass := r.FormValue("password_field")
		csrf := r.FormValue("csrf")

		if r.Method == "GET" {
			user = r.URL.Query().Get("username_field")
			pass = r.URL.Query().Get("password_field")
			csrf = r.URL.Query().Get("csrf")
		}

		if user == m.validUser && pass == m.validPass && csrf == "mock-csrf-token" {
			m.isOnline = true
			w.WriteHeader(200)
			fmt.Fprint(w, "<h1>Logged In!</h1>")
			return
		}
		w.WriteHeader(401)
		fmt.Fprint(w, "<h1>Unauthorized</h1>")

	default:
		w.WriteHeader(404)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestIntegration_FullFlow_POST(t *testing.T) {
	mock := &mockPortal{isOnline: false, validUser: "student123", validPass: "securepass"}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Probe: should detect portal
	p := prober.NewProber(client, logger)
	p.SetEndpoints([]prober.Endpoint{
		{
			URL: srv.URL + "/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "Success")
			},
		},
	})

	result := p.Check(context.Background())
	if result.Online {
		t.Fatal("expected offline, got online")
	}
	if !strings.Contains(result.FinalURL, "/login") {
		t.Fatalf("expected final URL with /login, got %q", result.FinalURL)
	}

	// 2. Fetch and parse portal page
	resp, err := client.Get(result.PortalURL)
	if err != nil {
		t.Fatalf("fetch portal: %v", err)
	}
	formData, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
	resp.Body.Close()
	if err != nil {
		t.Fatalf("parse form: %v", err)
	}

	if formData.UsernameField != "username_field" {
		t.Errorf("username field: want 'username_field', got %q", formData.UsernameField)
	}
	if formData.PasswordField != "password_field" {
		t.Errorf("password field: want 'password_field', got %q", formData.PasswordField)
	}
	if formData.Fields["csrf"] != "mock-csrf-token" {
		t.Errorf("csrf: want 'mock-csrf-token', got %q", formData.Fields["csrf"])
	}

	// 3. Submit login
	creds := &credential.Credentials{
		Username:      "student123",
		Password:      "securepass",
		UsernameField: formData.UsernameField,
		PasswordField: formData.PasswordField,
	}

	sub := auth.NewSubmitter(client, logger)
	if err := sub.Submit(context.Background(), formData, creds); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if !mock.isOnline {
		t.Fatal("portal should be online after successful login")
	}

	// 4. Re-probe to verify
	result2 := p.Check(context.Background())
	if !result2.Online {
		t.Error("expected online after login")
	}
}

func TestIntegration_AlreadyOnline(t *testing.T) {
	mock := &mockPortal{isOnline: true}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	p := prober.NewProber(client, logger)
	p.SetEndpoints([]prober.Endpoint{
		{
			URL: srv.URL + "/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "Success")
			},
		},
	})

	result := p.Check(context.Background())
	if !result.Online {
		t.Error("expected online=true")
	}
}

func TestIntegration_CredentialStore_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := credential.NewFileStore(filepath.Join(dir, "creds.json"))

	creds := &credential.Credentials{
		SSID:          "IntegrationTest_WiFi",
		Username:      "testuser",
		Password:      "testpass",
		UsernameField: "user",
		PasswordField: "pass",
		FormAction:    "http://portal/auth",
		FormMethod:    "POST",
		StaticFields:  map[string]string{"csrf": "abc"},
	}

	if err := store.Save(creds); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load("IntegrationTest_WiFi")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Username != "testuser" || loaded.Password != "testpass" {
		t.Error("credentials mismatch after roundtrip")
	}
}

func TestIntegration_WrongCredentials(t *testing.T) {
	mock := &mockPortal{isOnline: false, validUser: "real", validPass: "correct"}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	formData := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "POST",
		Fields:        map[string]string{"csrf": "mock-csrf-token"},
		UsernameField: "username_field",
		PasswordField: "password_field",
	}
	creds := &credential.Credentials{
		Username:      "wrong",
		Password:      "wrong",
		UsernameField: "username_field",
		PasswordField: "password_field",
	}

	sub := auth.NewSubmitter(client, logger)
	err := sub.Submit(context.Background(), formData, creds)
	if err != nil {
		t.Fatalf("submit should not error (server returned 401): %v", err)
	}

	if mock.isOnline {
		t.Error("portal should NOT be online with wrong credentials")
	}
}
