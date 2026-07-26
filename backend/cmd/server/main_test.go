package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	pkisvc "certflow/backend/internal/service/pki"

	"github.com/robfig/cron/v3"
)

func TestRunReturnsFailureForInvalidConfiguration(t *testing.T) {
	t.Setenv("PKI_KEY_ENCRYPTION_KEY", "invalid")

	if code := run(); code != 1 {
		t.Fatalf("run() exit code = %d, want 1", code)
	}
}

func TestExpiryReminderRunAtUsesCurrentCronMinute(t *testing.T) {
	schedule, err := cron.ParseStandard("0 0 * * *")
	if err != nil {
		t.Fatalf("parse cron: %v", err)
	}
	now := time.Date(2026, 7, 25, 0, 0, 24, 0, time.Local)
	if got := expiryReminderRunAt(schedule, now); !got.Equal(time.Date(2026, 7, 25, 0, 0, 0, 0, time.Local)) {
		t.Fatalf("expiryReminderRunAt() = %s, want current daily boundary", got)
	}
}

func TestExpiryScanAuditDetailUsesCertificateRenewalSummary(t *testing.T) {
	detail := expiryScanAuditDetail("0 0 * * *", 3, pkisvc.AutoRenewalReport{
		Enabled:              true,
		Renewed:              2,
		Failed:               1,
		NotificationFailures: 1,
	}, errors.New("renewal failed"), errors.New("reminder failed"))
	for _, expected := range []string{
		"扫描计划 0 0 * * * 已执行",
		"到期提醒已发送 3 次",
		"自动续期执行异常",
		"续期站外通知失败 1 次",
		"到期提醒发送异常",
	} {
		if !strings.Contains(detail, expected) {
			t.Fatalf("audit detail %q does not contain %q", detail, expected)
		}
	}
}
