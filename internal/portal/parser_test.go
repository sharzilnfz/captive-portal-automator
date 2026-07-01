package portal

import (
	"strings"
	"testing"
)

func TestParseLoginForm_Basic(t *testing.T) {
	h := `<html><form action="/auth" method="POST">
		<input type="hidden" name="csrf" value="token123">
		<input type="text" name="username_field">
		<input type="password" name="password_field">
	</form></html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/auth" {
		t.Errorf("action: want 'http://localhost:8080/auth', got %q", fd.Action)
	}
	if fd.Method != "POST" {
		t.Errorf("method: want POST, got %q", fd.Method)
	}
	if fd.UsernameField != "username_field" {
		t.Errorf("username: want 'username_field', got %q", fd.UsernameField)
	}
	if fd.PasswordField != "password_field" {
		t.Errorf("password: want 'password_field', got %q", fd.PasswordField)
	}
	if fd.Fields["csrf"] != "token123" {
		t.Errorf("csrf: want 'token123', got %q", fd.Fields["csrf"])
	}
}

func TestParseLoginForm_NoAction_GET(t *testing.T) {
	h := `<form method="GET">
		<input type="text" name="email_addr">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/login" {
		t.Errorf("action should fallback to base URL, got %q", fd.Action)
	}
	if fd.Method != "GET" {
		t.Errorf("method: want GET, got %q", fd.Method)
	}
	if fd.UsernameField != "email_addr" {
		t.Errorf("username: want 'email_addr', got %q", fd.UsernameField)
	}
}

func TestParseLoginForm_MultiForm_PrioritizesPassword(t *testing.T) {
	h := `<html>
		<form action="/lang" method="GET">
			<input type="submit" name="lang" value="en">
		</form>
		<form action="/login-submit" method="POST">
			<input type="text" name="email">
			<input type="password" name="pass">
		</form>
	</html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/login-submit" {
		t.Errorf("should pick login form, got action %q", fd.Action)
	}
	if fd.PasswordField != "pass" {
		t.Errorf("password: want 'pass', got %q", fd.PasswordField)
	}
}

func TestParseLoginForm_FallbackUsername(t *testing.T) {
	h := `<form action="/login" method="POST">
		<input type="hidden" name="csrf" value="xyz">
		<input type="text" name="weird_field">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "weird_field" {
		t.Errorf("fallback username: want 'weird_field', got %q", fd.UsernameField)
	}
}

func TestParseLoginForm_PhoneKeyword(t *testing.T) {
	h := `<form action="/login" method="POST">
		<input type="text" name="phone">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "phone" {
		t.Errorf("username: want 'phone', got %q", fd.UsernameField)
	}
}

func TestParseLoginForm_NoForm(t *testing.T) {
	h := `<html><body><p>No form here</p></body></html>`
	_, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/login")
	if err == nil {
		t.Error("expected error for page with no form")
	}
}

func TestParseLoginForm_AbsoluteAction(t *testing.T) {
	h := `<form action="https://auth.example.com/submit" method="POST">
		<input type="text" name="user">
		<input type="password" name="pwd">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://portal.example.com/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "https://auth.example.com/submit" {
		t.Errorf("absolute action should be preserved, got %q", fd.Action)
	}
}

func TestParseLoginForm_HiddenFields(t *testing.T) {
	h := `<form action="/auth" method="POST">
		<input type="hidden" name="csrf" value="abc">
		<input type="hidden" name="session" value="xyz">
		<input type="hidden" name="redirect" value="/dashboard">
		<input type="text" name="user">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Fields["csrf"] != "abc" {
		t.Errorf("csrf: want 'abc', got %q", fd.Fields["csrf"])
	}
	if fd.Fields["session"] != "xyz" {
		t.Errorf("session: want 'xyz', got %q", fd.Fields["session"])
	}
	if fd.Fields["redirect"] != "/dashboard" {
		t.Errorf("redirect: want '/dashboard', got %q", fd.Fields["redirect"])
	}
}
