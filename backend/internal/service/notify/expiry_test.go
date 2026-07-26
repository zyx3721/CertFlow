package notify

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestExpiryNotificationDaysUsesConfiguredValueAndFallback(t *testing.T) {
	if got := expiryNotificationDays(map[string]any{"expiryNotificationDays": float64(21)}); got != 21 {
		t.Fatalf("expiryNotificationDays() = %d, want 21", got)
	}
	if got := expiryNotificationDays(map[string]any{"expiryNotificationDays": float64(0)}); got != 15 {
		t.Fatalf("expiryNotificationDays() = %d, want fallback 15", got)
	}
}

func TestDefaultExpiryNotificationCronIsValid(t *testing.T) {
	if _, err := cron.ParseStandard(defaultExpiryNotificationCron); err != nil {
		t.Fatalf("default expiry notification cron must be valid: %v", err)
	}
}

func TestRenewalNotificationMessage(t *testing.T) {
	notAfter := time.Date(2027, 7, 25, 16, 30, 45, 0, time.Local)
	if got := renewalNotificationMessage("test", "example.com", &notAfter); got != "用户 test 申请的证书 example.com 自动续期成功，将于 2027-07-25 16:30:45 后过期" {
		t.Fatalf("success renewal message = %q", got)
	}
	if got := renewalNotificationMessage("test", "example.com", nil); got != "用户 test 申请的证书 example.com 自动续期失败，请前往平台检查日志" {
		t.Fatalf("failure renewal message = %q", got)
	}
}
