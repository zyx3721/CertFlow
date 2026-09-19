package auth

import (
	"strings"
	"testing"
	"time"
)

func TestSignWecomStateRoundTrip(t *testing.T) {
	secret := "unit-test-secret"
	signed, err := signWecomState(secret, wecomState{Purpose: "login", UserID: ""}, defaultWecomStateTTL)
	if err != nil {
		t.Fatalf("signWecomState failed: %v", err)
	}
	state, err := verifyWecomState(secret, signed, "login")
	if err != nil {
		t.Fatalf("verifyWecomState failed: %v", err)
	}
	if state.Purpose != "login" {
		t.Fatalf("purpose mismatch: %s", state.Purpose)
	}
	if state.Exp <= time.Now().Unix() {
		t.Fatalf("state should not be expired yet: %d", state.Exp)
	}
}

func TestVerifyWecomStateRejectsTamperedSignature(t *testing.T) {
	secret := "unit-test-secret"
	signed, err := signWecomState(secret, wecomState{Purpose: "login"}, defaultWecomStateTTL)
	if err != nil {
		t.Fatalf("signWecomState failed: %v", err)
	}
	if _, err := verifyWecomState(secret, signed+"x", "login"); err == nil {
		t.Fatal("tampered signature should be rejected")
	}
	if _, err := verifyWecomState("another-secret", signed, "login"); err == nil {
		t.Fatal("wrong secret should be rejected")
	}
}

func TestVerifyWecomStateRejectsPurposeMismatch(t *testing.T) {
	secret := "unit-test-secret"
	signed, err := signWecomState(secret, wecomState{Purpose: "login", UserID: "user-1"}, defaultWecomStateTTL)
	if err != nil {
		t.Fatalf("signWecomState failed: %v", err)
	}
	if _, err := verifyWecomState(secret, signed, "bind"); err == nil {
		t.Fatal("purpose mismatch should be rejected")
	}
}

func TestVerifyWecomStateRejectsExpired(t *testing.T) {
	secret := "unit-test-secret"
	expired, err := signWecomStatePayload(secret, wecomState{Purpose: "login", Exp: time.Now().Add(-time.Minute).Unix()})
	if err != nil {
		t.Fatalf("signWecomStatePayload failed: %v", err)
	}
	if _, err := verifyWecomState(secret, expired, "login"); err == nil {
		t.Fatal("expired state should be rejected")
	}
}

func TestWecomConfigValidate(t *testing.T) {
	direct := WecomConfig{Mode: WecomModeDirect, CorpID: "ww1", AgentID: "1000002", Secret: "s"}
	if err := direct.validate(); err != nil {
		t.Fatalf("valid direct config should pass: %v", err)
	}
	missing := direct
	missing.Secret = ""
	if err := missing.validate(); err == nil {
		t.Fatal("direct config without secret should fail")
	}
	sso := WecomConfig{Mode: WecomModeSSO, SSOBaseURL: "https://sso.example.com", SSOAppID: "certflow", SSOAppSecret: strings.Repeat("x", 32)}
	if err := sso.validate(); err != nil {
		t.Fatalf("valid sso config should pass: %v", err)
	}
	badURL := sso
	badURL.SSOBaseURL = "sso.example.com"
	if err := badURL.validate(); err == nil {
		t.Fatal("sso base url without scheme should fail")
	}
	unknown := WecomConfig{Mode: "other"}
	if err := unknown.validate(); err == nil {
		t.Fatal("unknown mode should fail")
	}
}

func TestNormalizeWecomMode(t *testing.T) {
	if normalizeWecomMode("sso") != WecomModeSSO {
		t.Fatal("sso should be preserved")
	}
	if normalizeWecomMode("direct") != WecomModeDirect {
		t.Fatal("direct should be preserved")
	}
	if normalizeWecomMode("") != WecomModeDirect {
		t.Fatal("empty should fall back to direct")
	}
	if normalizeWecomMode(123) != WecomModeDirect {
		t.Fatal("non-string should fall back to direct")
	}
}

func TestWecomCallbackRedirectURL(t *testing.T) {
	cfg := WecomConfig{RedirectPrefix: "https://cert.example.com/prefix/"}
	if got := cfg.wecomCallbackRedirectURL("http", "local:8080"); got != "https://cert.example.com/prefix/login" {
		t.Fatalf("unexpected redirect: %s", got)
	}
	fallback := WecomConfig{}
	if got := fallback.wecomCallbackRedirectURL("https", "cert.example.com"); got != "https://cert.example.com/login" {
		t.Fatalf("unexpected redirect: %s", got)
	}
}
