package portal

import (
	"errors"
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

func TestParseLoginForm_IframeRedirect(t *testing.T) {
	h := `<html><body>
		<iframe src="/real-login"></iframe>
	</body></html>`

	_, err := ParseLoginForm(strings.NewReader(h), "http://wifi.example.com/portal")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var redir *ErrRedirect
	if !errors.As(err, &redir) {
		t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
	}
	want := "http://wifi.example.com/real-login"
	if redir.URL != want {
		t.Errorf("redirect URL: want %q, got %q", want, redir.URL)
	}
}

func TestParseLoginForm_JSRedirect(t *testing.T) {
	h := `<html><head>
		<script>window.location = '/portal';</script>
	</head><body></body></html>`

	_, err := ParseLoginForm(strings.NewReader(h), "http://wifi.example.com/gate")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var redir *ErrRedirect
	if !errors.As(err, &redir) {
		t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
	}
	want := "http://wifi.example.com/portal"
	if redir.URL != want {
		t.Errorf("redirect URL: want %q, got %q", want, redir.URL)
	}
}

func TestParseLoginForm_MetaRefreshNoForm(t *testing.T) {
	h := `<html><head>
		<meta http-equiv="refresh" content="0;url=/login">
	</head><body></body></html>`

	_, err := ParseLoginForm(strings.NewReader(h), "http://wifi.example.com/portal")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var redir *ErrRedirect
	if !errors.As(err, &redir) {
		t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
	}
	want := "http://wifi.example.com/login"
	if redir.URL != want {
		t.Errorf("redirect URL: want %q, got %q", want, redir.URL)
	}
}

// TestParseLoginForm_RuijieIDInputs exercises the BRACU/Ruijie portal pattern:
// no <form> element, inputs identified by id="username" / id="password".
func TestParseLoginForm_RuijieIDInputs(t *testing.T) {
	h := `<!DOCTYPE html>
<html>
 <head>
  <script type="text/javascript" src="/material/custom/auth.js"></script>
 </head>
 <body>
  <div authpluginarray="1,2,5" modelnum="0">
   <input id="username" value="" placeholder="Username" type="text" class="frm-input">
   <input id="password" value="" placeholder="Password" type="password" autocomplete="off" class="frm-input">
   <input id="loginBtn" type="button" value="Log In" onclick="login();">
  </div>
 </body>
</html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "https://wifi2.example.com:19008/portal?ap-mac=aabbcc&ssid=Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "username" {
		t.Errorf("UsernameField: want %q, got %q", "username", fd.UsernameField)
	}
	if fd.PasswordField != "password" {
		t.Errorf("PasswordField: want %q, got %q", "password", fd.PasswordField)
	}
	if fd.Method != "POST" {
		t.Errorf("Method: want POST, got %q", fd.Method)
	}
	if fd.Action == "" {
		t.Error("Action must not be empty")
	}
}

// TestParseLoginForm_RuijieIDInputs_OnlyText confirms that a page with
// only a text input and no password field is not synthesised → ErrNoForm.
// synthesizeFormFromIDs requires a password field to avoid false positives.
func TestParseLoginForm_RuijieIDInputs_OnlyText(t *testing.T) {
	h := `<html><body>
		<input id="userId" type="text" placeholder="Student ID">
	</body></html>`

	_, err := ParseLoginForm(strings.NewReader(h), "http://portal.example.com/")
	if err == nil {
		t.Fatal("expected ErrNoForm when no password field present")
	}
}



// TestIsRuijiePortal verifies HTML fingerprinting.
func TestIsRuijiePortal(t *testing.T) {
	h := `<html><body>
		<div authpluginarray="1,2,5" modelnum="0"></div>
	</body></html>`

	doc, err := ParseDoc(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !IsRuijiePortal(doc) {
		t.Error("IsRuijiePortal: expected true for Ruijie fingerprint HTML")
	}
}

// TestRuijieLoginURL confirms canonical URL construction.
func TestRuijieLoginURL(t *testing.T) {
	got := RuijieLoginURL("https://wifi2.bracu.ac.bd:19008/portal?ap-mac=abc&ssid=Test")
	want := "https://wifi2.bracu.ac.bd:19008/eportal/InterFace.do?method=login"
	if got != want {
		t.Errorf("RuijieLoginURL: want %q, got %q", want, got)
	}
}

// TestSynthesizeRuijieForm_JSShell verifies that a Ruijie page with only
// script tags and div markers (no <input> elements) produces a valid FormData.
func TestSynthesizeRuijieForm_JSShell(t *testing.T) {
	h := `<!DOCTYPE html>
<html>
 <head>
  <script type="text/javascript" src="/material/custom/auth.js"></script>
  <script type="text/javascript" src="/eportal/portal/custom/js/portal.js"></script>
 </head>
 <body>
  <div authpluginarray="1,2,5" modelnum="0">
   <!-- Login form rendered by JavaScript -->
  </div>
 </body>
</html>`

	doc, err := ParseDoc(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	fd := SynthesizeRuijieForm(doc, "https://wifi2.bracu.ac.bd:19008/portal?ap-mac=aabbcc&ssid=Test")
	if fd == nil {
		t.Fatal("SynthesizeRuijieForm returned nil for Ruijie fingerprint HTML")
	}
	if fd.UsernameField != "userId" {
		t.Errorf("UsernameField: want %q, got %q", "userId", fd.UsernameField)
	}
	if fd.PasswordField != "password" {
		t.Errorf("PasswordField: want %q, got %q", "password", fd.PasswordField)
	}
	if fd.Method != "POST" {
		t.Errorf("Method: want POST, got %q", fd.Method)
	}
	want := "https://wifi2.bracu.ac.bd:19008/eportal/InterFace.do?method=login"
	if fd.Action != want {
		t.Errorf("Action: want %q, got %q", want, fd.Action)
	}
	if fd.Fields["passwordEncrypt"] != "false" {
		t.Errorf("passwordEncrypt: want %q, got %q", "false", fd.Fields["passwordEncrypt"])
	}
}

// TestSynthesizeRuijieForm_NotRuijie confirms non-Ruijie pages return nil.
func TestSynthesizeRuijieForm_NotRuijie(t *testing.T) {
	h := `<html><body><p>Generic page</p></body></html>`

	doc, err := ParseDoc(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	fd := SynthesizeRuijieForm(doc, "http://portal.example.com/login")
	if fd != nil {
		t.Error("SynthesizeRuijieForm should return nil for non-Ruijie HTML")
	}
}

// TestSynthesizeRuijieForm_EportalScript confirms the /eportal/ script src
// fingerprint is detected even without authpluginarray attributes.
func TestSynthesizeRuijieForm_EportalScript(t *testing.T) {
	h := `<!DOCTYPE html>
<html>
 <head>
  <script src="/eportal/portal/custom/js/main.js"></script>
 </head>
 <body>
  <div id="app"></div>
 </body>
</html>`

	doc, err := ParseDoc(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	fd := SynthesizeRuijieForm(doc, "https://wifi2.bracu.ac.bd:19008/portal?ssid=Test")
	if fd == nil {
		t.Fatal("SynthesizeRuijieForm returned nil for /eportal/ script fingerprint")
	}
	if fd.Action != "https://wifi2.bracu.ac.bd:19008/eportal/InterFace.do?method=login" {
		t.Errorf("Action: unexpected %q", fd.Action)
	}
}

// TestParseLoginForm_ClickThrough verifies that a form with no username and
// password inputs (click-through) is parsed successfully and picked.
func TestParseLoginForm_ClickThrough(t *testing.T) {
	h := `<html>
		<form action="/connect" method="POST">
			<input type="hidden" name="client_mac" value="11:22:33:44:55:66">
			<input type="submit" value="Connect Free WiFi">
		</form>
	</html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/portal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "" {
		t.Errorf("expected empty UsernameField, got %q", fd.UsernameField)
	}
	if fd.PasswordField != "" {
		t.Errorf("expected empty PasswordField, got %q", fd.PasswordField)
	}
	if fd.Fields["client_mac"] != "11:22:33:44:55:66" {
		t.Errorf("expected client_mac field, got %q", fd.Fields["client_mac"])
	}
}

// TestParseLoginForm_Checkboxes verifies checkbox fields default to "on"
// if they have no explicit value.
func TestParseLoginForm_Checkboxes(t *testing.T) {
	h := `<html>
		<form action="/login" method="POST">
			<input type="checkbox" name="agree_terms">
			<input type="checkbox" name="subscribe" value="yes">
			<input type="password" name="pwd">
		</form>
	</html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Fields["agree_terms"] != "on" {
		t.Errorf("agree_terms checkbox: expected 'on', got %q", fd.Fields["agree_terms"])
	}
	if fd.Fields["subscribe"] != "yes" {
		t.Errorf("subscribe checkbox: expected 'yes', got %q", fd.Fields["subscribe"])
	}
}

// TestParseLoginForm_RadioButtons verifies that the checked radio button
// value is preferred over the others.
func TestParseLoginForm_RadioButtons(t *testing.T) {
	h := `<html>
		<form action="/login" method="POST">
			<input type="radio" name="auth_type" value="sms">
			<input type="radio" name="auth_type" value="voucher" checked>
			<input type="radio" name="auth_type" value="free">
			<input type="password" name="pwd">
		</form>
	</html>`

	fd, err := ParseLoginForm(strings.NewReader(h), "http://localhost:8080/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Fields["auth_type"] != "voucher" {
		t.Errorf("radio button auth_type: expected 'voucher' (checked), got %q", fd.Fields["auth_type"])
	}
}

