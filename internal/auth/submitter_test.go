package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/portal"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestSubmitter_POST(t *testing.T) {
	var receivedUser, receivedPass, receivedCSRF string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/auth" {
			r.ParseForm()
			receivedUser = r.FormValue("user")
			receivedPass = r.FormValue("pass")
			receivedCSRF = r.FormValue("csrf")
			w.WriteHeader(200)
			fmt.Fprint(w, "OK")
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sub := NewSubmitter(client, testLogger())

	form := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "POST",
		Fields:        map[string]string{"csrf": "token123"},
		UsernameField: "user",
		PasswordField: "pass",
	}
	creds := &credential.Credentials{
		Username:      "student",
		Password:      "secret",
		UsernameField: "user",
		PasswordField: "pass",
	}

	err := sub.Submit(context.Background(), form, creds)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if receivedUser != "student" {
		t.Errorf("user: want 'student', got %q", receivedUser)
	}
	if receivedPass != "secret" {
		t.Errorf("pass: want 'secret', got %q", receivedPass)
	}
	if receivedCSRF != "token123" {
		t.Errorf("csrf: want 'token123', got %q", receivedCSRF)
	}
}

func TestSubmitter_GET(t *testing.T) {
	var receivedUser, receivedPass string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/auth" {
			receivedUser = r.URL.Query().Get("user")
			receivedPass = r.URL.Query().Get("pass")
			w.WriteHeader(200)
			fmt.Fprint(w, "OK")
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sub := NewSubmitter(client, testLogger())

	form := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "GET",
		Fields:        map[string]string{},
		UsernameField: "user",
		PasswordField: "pass",
	}
	creds := &credential.Credentials{
		Username:      "student",
		Password:      "secret",
		UsernameField: "user",
		PasswordField: "pass",
	}

	err := sub.Submit(context.Background(), form, creds)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if receivedUser != "student" {
		t.Errorf("user: want 'student', got %q", receivedUser)
	}
	if receivedPass != "secret" {
		t.Errorf("pass: want 'secret', got %q", receivedPass)
	}
}
