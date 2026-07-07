package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sharzilnafis/autocap/internal/credential"
	"github.com/sharzilnafis/autocap/internal/portal"
	"github.com/sharzilnafis/autocap/internal/prober"
)

// Submitter handles login form submission and verification.
type Submitter struct {
	client *http.Client
	logger *slog.Logger
}

// NewSubmitter creates a Submitter using the shared HTTP client.
func NewSubmitter(client *http.Client, logger *slog.Logger) *Submitter {
	return &Submitter{client: client, logger: logger}
}

// Submit sends the login form with credentials.
func (s *Submitter) Submit(ctx context.Context, form *portal.FormData, creds *credential.Credentials) error {
	payload := url.Values{}
	for k, v := range form.Fields {
		payload.Set(k, v)
	}

	// Use credential field names if set, otherwise use form-detected field names
	usernameField := creds.UsernameField
	if usernameField == "" {
		usernameField = form.UsernameField
	}
	passwordField := creds.PasswordField
	if passwordField == "" {
		passwordField = form.PasswordField
	}

	if usernameField != "" {
		payload.Set(usernameField, creds.Username)
	}
	if passwordField != "" {
		payload.Set(passwordField, creds.Password)
	}

	// Debug: log what we're about to send (redact password).
	debugFields := make([]string, 0, len(payload))
	for k := range payload {
		if k == passwordField {
			debugFields = append(debugFields, k+"=***")
		} else {
			debugFields = append(debugFields, k+"="+payload.Get(k))
		}
	}
	s.logger.Debug("login payload", "fields", debugFields)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	method := form.Method
	if method == "" {
		method = "POST"
	}

	var req *http.Request
	var err error

	if method == "GET" {
		u, parseErr := url.Parse(form.Action)
		if parseErr != nil {
			return fmt.Errorf("auth: parse action URL: %w", parseErr)
		}
		q := u.Query()
		for k, vs := range payload {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	} else {
		body := payload.Encode()
		req, err = http.NewRequestWithContext(ctx, "POST", form.Action, strings.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return fmt.Errorf("auth: create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	s.logger.Info("submitting login", "action", form.Action, "method", method)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: submit: %w", err)
	}
	defer resp.Body.Close()

	// Read and log the response body (portals often return error details).
	respBody, _ := io.ReadAll(resp.Body)
	s.logger.Info("login submitted", "status", resp.StatusCode)
	if len(respBody) > 0 {
		snip := string(respBody)
		if len(snip) > 1000 {
			snip = snip[:1000]
		}
		s.logger.Debug("login response body", "body", snip)
	}

	return nil
}

// Verify re-probes to confirm internet access after login.
func (s *Submitter) Verify(ctx context.Context, p *prober.Prober) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(2500 * time.Millisecond):
	}

	result := p.Check(ctx)
	return result.Online, nil
}
