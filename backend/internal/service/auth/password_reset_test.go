package auth

import (
	"testing"
	"time"
)

func TestMaskEmail(t *testing.T) {
	tests := map[string]string{
		"admin@example.com": "a****@example.com",
		"ab@example.com":    "a**@example.com",
		"invalid":           "***",
	}
	for value, expected := range tests {
		if actual := maskEmail(value); actual != expected {
			t.Fatalf("maskEmail(%q) = %q, want %q", value, actual, expected)
		}
	}
}

func TestResetCaptchaSignature(t *testing.T) {
	service := &Service{captchaSecret: []byte("test-secret")}
	token := service.signResetCaptcha("12", time.Now().Add(time.Minute))
	if !service.verifyResetCaptcha(token, "12") {
		t.Fatal("expected signed captcha to verify")
	}
	if service.verifyResetCaptcha(token, "13") {
		t.Fatal("wrong captcha answer must not verify")
	}
	if service.verifyResetCaptcha(token+"x", "12") {
		t.Fatal("tampered captcha token must not verify")
	}
	if service.verifyResetCaptcha(service.signResetCaptcha("12", time.Now().Add(-time.Minute)), "12") {
		t.Fatal("expired captcha token must not verify")
	}
}

func TestInvalidResetCaptchaMessageMatchesResetFlow(t *testing.T) {
	if ErrInvalidResetCaptcha.Error() != "验证码不正确" {
		t.Fatalf("unexpected reset captcha message: %q", ErrInvalidResetCaptcha.Error())
	}
}

func TestMinutesDurationUsesBoundsAndFallback(t *testing.T) {
	if actual := minutesDuration(0.5, 1, 0.5, 10); actual != 30*time.Second {
		t.Fatalf("minutesDuration() = %s, want 30s", actual)
	}
	if actual := minutesDuration(99, 5, 1, 10); actual != 5*time.Minute {
		t.Fatalf("out-of-range duration = %s, want 5m", actual)
	}
	if actual := minutesDuration("invalid", 5, 1, 10); actual != 5*time.Minute {
		t.Fatalf("invalid duration = %s, want 5m", actual)
	}
}

func TestDefaultPasswordResetConfig(t *testing.T) {
	config := defaultPasswordResetConfig()
	if config.resetCaptchaTTL != time.Minute || config.resetCooldown != 30*time.Second {
		t.Fatalf("defaults = %#v", config)
	}
}
