package portal

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Form action URL resolution (relative / absolute / scheme-relative / missing)
// ─────────────────────────────────────────────────────────────────────────────

func TestParseLoginForm_ActionResolution(t *testing.T) {
	cases := []struct {
		name   string
		action string // raw attribute; empty string omits the attribute entirely
		base   string
		want   string
	}{
		{"relative path", `action="/auth"`, "http://portal.example.com/login", "http://portal.example.com/auth"},
		{"relative no leading slash", `action="auth"`, "http://portal.example.com/dir/login", "http://portal.example.com/dir/auth"},
		{"absolute URL", `action="http://other.example.com/x"`, "http://portal.example.com/login", "http://other.example.com/x"},
		{"scheme-relative", `action="//cdn.example.com/auth"`, "http://portal.example.com/login", "http://cdn.example.com/auth"},
		{"missing action submits to base", "", "http://portal.example.com/login", "http://portal.example.com/login"},
		{"action keeps query string", `action="/eportal/InterFace.do?method=login"`,
			"https://wifi.example.com:19008/portal?ap-mac=aabb", "https://wifi.example.com:19008/eportal/InterFace.do?method=login"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := fmt.Sprintf(`<html><body>
<form method="post" %s>
  <input type="text" name="user">
  <input type="password" name="pass">
</form></body></html>`, tc.action)
			fd, err := ParseLoginForm(strings.NewReader(h), tc.base)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if fd.Action != tc.want {
				t.Errorf("Action = %q, want %q", fd.Action, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vendor form shapes: MikroTik, pfSense, Cisco ISE
// ─────────────────────────────────────────────────────────────────────────────

func TestParseLoginForm_MikroTik_HiddenDstPopup(t *testing.T) {
	h := `<html><body>
<form name="login" action="/login" method="post">
  <input type="text" name="username">
  <input type="password" name="password">
  <input type="hidden" name="dst" value="http://detectportal.firefox.com/">
  <input type="hidden" name="popup" value="true">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://192.168.88.1/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.UsernameField != "username" || fd.PasswordField != "password" {
		t.Errorf("fields = %q/%q, want username/password", fd.UsernameField, fd.PasswordField)
	}
	if fd.Fields["dst"] != "http://detectportal.firefox.com/" {
		t.Errorf("dst = %q, hidden dst must be preserved", fd.Fields["dst"])
	}
	if fd.Fields["popup"] != "true" {
		t.Errorf("popup = %q, want \"true\"", fd.Fields["popup"])
	}
	if fd.Action != "http://192.168.88.1/login" {
		t.Errorf("Action = %q", fd.Action)
	}
}

func TestParseLoginForm_MikroTik_ImageSubmitButton(t *testing.T) {
	// [POST-FIX] MikroTik trial portals submit via <input type="image">,
	// which browsers encode as name.x / name.y. extractFormData must emit these.
	h := `<html><body>
<form action="/login" method="post">
  <input type="hidden" name="mac" value="aa:bb:cc:dd:ee:ff">
  <input type="image" name="accept" src="button.gif" alt="log in">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://192.168.88.1/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.Fields["accept.x"] == "" || fd.Fields["accept.y"] == "" {
		t.Errorf("image submit button not encoded: Fields = %v", fd.Fields)
	}
}

func TestParseLoginForm_PfSense_AuthUserZone(t *testing.T) {
	h := `<html><body>
<form method="post" action="index.php">
  <input name="auth_user" type="text">
  <input name="auth_pass" type="password">
  <input name="redirurl" type="hidden" value="http://captive.apple.com/hotspot-detect.html">
  <input name="zone" type="hidden" value="Voucher Tests">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://192.168.1.1:8002/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.UsernameField != "auth_user" {
		t.Errorf("UsernameField = %q, want auth_user", fd.UsernameField)
	}
	if fd.PasswordField != "auth_pass" {
		t.Errorf("PasswordField = %q, want auth_pass", fd.PasswordField)
	}
	if fd.Fields["zone"] != "Voucher Tests" {
		t.Errorf("zone = %q, hidden zone must be preserved verbatim", fd.Fields["zone"])
	}
	if fd.Fields["redirurl"] == "" {
		t.Error("redirurl hidden field missing")
	}
	if fd.Action != "http://192.168.1.1:8002/index.php" {
		t.Errorf("Action = %q", fd.Action)
	}
}

func TestParseLoginForm_CiscoISE_ViewStatePreserved(t *testing.T) {
	// javax.faces.ViewState contains ':' and '=' — must round-trip unmodified.
	h := `<html><body>
<form id="login" method="post" action="/login.do">
  <input type="hidden" name="javax.faces.ViewState" value="-7766554433221100:AbCdEfGh==">
  <input type="text" name="user">
  <input type="password" name="password">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "https://ise.example.com/login")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := "-7766554433221100:AbCdEfGh=="
	if got := fd.Fields["javax.faces.ViewState"]; got != want {
		t.Errorf("ViewState = %q, want %q", got, want)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Input-handling edge cases
// ─────────────────────────────────────────────────────────────────────────────

func TestParseLoginForm_MethodNormalizedToUpper(t *testing.T) {
	// [POST-FIX] a lowercase method="get" must not be misread as POST by Submit.
	h := `<form action="/auth" method="get">
<input type="text" name="user"><input type="password" name="pass"></form>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.Method != "GET" {
		t.Errorf("Method = %q, want GET (normalized)", fd.Method)
	}
}

func TestParseLoginForm_DisabledInputsSkipped(t *testing.T) {
	// [POST-FIX] disabled inputs are not submitted by browsers.
	h := `<form action="/auth" method="post">
<input type="hidden" name="tracking" value="xyz" disabled>
<input type="hidden" name="csrf" value="tok">
<input type="text" name="user">
<input type="password" name="pass">
</form>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := fd.Fields["tracking"]; ok {
		t.Error("disabled input must not be captured")
	}
	if fd.Fields["csrf"] != "tok" {
		t.Errorf("csrf = %q, want tok", fd.Fields["csrf"])
	}
}

func TestParseLoginForm_FormAttributeAssociation(t *testing.T) {
	// [POST-FIX] HTML5: an input outside the <form> linked via form="id"
	// belongs to that form. Requires a document-level scan.
	h := `<html><body>
<form id="loginform" action="/auth" method="post">
  <input type="text" name="user">
</form>
<input type="password" name="pw_outside" form="loginform">
</body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.PasswordField != "pw_outside" {
		t.Errorf("PasswordField = %q, want pw_outside (form= association)", fd.PasswordField)
	}
}

func TestParseLoginForm_DuplicateHiddenNames(t *testing.T) {
	h := `<form action="/auth" method="post">
<input type="hidden" name="dup" value="v1">
<input type="hidden" name="dup" value="v2">
<input type="text" name="user">
<input type="password" name="pass">
</form>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, ok := fd.Fields["dup"]
	if !ok || (v != "v1" && v != "v2") {
		t.Errorf("dup = %q (present=%v); expected one of v1/v2", v, ok)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Form selection / scoring
// ─────────────────────────────────────────────────────────────────────────────

func TestParseLoginForm_MultipleForms_PicksLogin(t *testing.T) {
	// A search box appears before the real login form in DOM order.
	h := `<html><body>
<form action="/search" method="get"><input type="text" name="q"></form>
<form action="/login" method="post">
  <input type="text" name="user">
  <input type="password" name="pass">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.PasswordField != "pass" {
		t.Errorf("picked wrong form: PasswordField = %q", fd.PasswordField)
	}
	if !strings.HasSuffix(fd.Action, "/login") {
		t.Errorf("picked wrong form: Action = %q", fd.Action)
	}
}

func TestParseLoginForm_UsernameKeywordDetection(t *testing.T) {
	names := []string{
		"auth_user", // pfSense
		"User",      // Aruba (capitalized)
		"loginId",
		"email",
		"phone_number",
		"member_name",
		"account_no",
		"telephone",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			h := fmt.Sprintf(`<form action="/auth" method="post">
<input type="text" name=%q>
<input type="password" name="pass">
</form>`, name)
			fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if fd.UsernameField != name {
				t.Errorf("UsernameField = %q, want %q", fd.UsernameField, name)
			}
		})
	}
}

func TestParseLoginForm_HiddenSessionIDNeverUsername(t *testing.T) {
	// Hidden inputs must never be promoted to the username field,
	// even when their name matches a keyword ("sessionid" contains "id").
	h := `<form action="/auth" method="post">
<input type="hidden" name="sessionid" value="abc123">
<input type="password" name="pass">
</form>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fd.UsernameField == "sessionid" {
		t.Error("hidden sessionid must not be detected as the username field")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Redirect-vs-form preference
// ─────────────────────────────────────────────────────────────────────────────

func TestParseLoginForm_JunkFormYieldsToRedirect(t *testing.T) {
	// [POST-FIX] a page with a useless form (no credentials, no hidden state)
	// plus a meta refresh should follow the redirect, not submit the junk form.
	h := `<html><head><meta http-equiv="refresh" content="0;url=/real-login"></head>
<body><form action="/search"><input type="submit" value="Search"></form></body></html>`
	_, err := ParseLoginForm(strings.NewReader(h), "http://x/entry")
	if err == nil {
		t.Fatal("expected ErrRedirect, got nil")
	}
	var redir *ErrRedirect
	if !errors.As(err, &redir) {
		t.Fatalf("expected *ErrRedirect, got %T: %v", err, err)
	}
	if redir.URL != "http://x/real-login" {
		t.Errorf("redirect URL = %q, want http://x/real-login", redir.URL)
	}
}

func TestParseLoginForm_ClickThroughSurvivesRedirectFix(t *testing.T) {
	// Regression guard: a click-through form WITH hidden state must still be
	// parsed even when a redirect hint is present on the same page.
	h := `<html><head><meta http-equiv="refresh" content="0;url=/other"></head>
<body><form action="/portal" method="post">
<input type="hidden" name="client_mac" value="11:22:33:44:55:66">
<input type="hidden" name="accept" value="true">
</form></body></html>`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/entry")
	if err != nil {
		t.Fatalf("click-through form must parse, got: %v", err)
	}
	if fd.Fields["client_mac"] != "11:22:33:44:55:66" {
		t.Errorf("client_mac = %q", fd.Fields["client_mac"])
	}
}

func TestParseLoginForm_MalformedHTMLNoPanic(t *testing.T) {
	h := `<form action=/auth method=post><input type=password name=p><input type=text name=u`
	fd, err := ParseLoginForm(strings.NewReader(h), "http://x/")
	// No assertion on the outcome — only that it does not panic.
	_ = fd
	_ = err
}
