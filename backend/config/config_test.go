package config

import (
	"encoding/base64"
	"io"
	"log/slog"
	"testing"
)

func TestLoadUsesConfiguredJWTSecret(t *testing.T) {
	setRequiredTestEnv(t)
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := Load(discardLogger())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.SessionSecret != "test-secret" {
		t.Fatalf("SessionSecret = %q, want configured value", cfg.Auth.SessionSecret)
	}
}

func TestLoadGeneratesTemporaryJWTSecret(t *testing.T) {
	setRequiredTestEnv(t)
	t.Setenv("JWT_SECRET", "")

	first, err := Load(discardLogger())
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	second, err := Load(discardLogger())
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if first.Auth.SessionSecret == "" {
		t.Fatal("SessionSecret is empty")
	}
	if first.Auth.SessionSecret == second.Auth.SessionSecret {
		t.Fatal("temporary SessionSecret was reused")
	}
}

func setRequiredTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PKI_KEY_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
