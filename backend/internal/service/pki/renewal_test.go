package pki

import "testing"

func TestRenewalDaysUsesConfiguredValueAndFallback(t *testing.T) {
	if got := renewalDays(map[string]any{"renewDays": float64(21)}); got != 21 {
		t.Fatalf("renewalDays() = %d, want 21", got)
	}
	if got := renewalDays(map[string]any{"renewDays": float64(0)}); got != defaultRenewDays {
		t.Fatalf("renewalDays() = %d, want fallback %d", got, defaultRenewDays)
	}
}

func TestAutoRenewEnabledRequiresBooleanTrue(t *testing.T) {
	if !autoRenewEnabled(map[string]any{"autoRenew": true}) {
		t.Fatal("boolean true must enable automatic renewal")
	}
	if autoRenewEnabled(map[string]any{"autoRenew": "true"}) {
		t.Fatal("non-boolean configuration must not enable automatic renewal")
	}
}

func TestAutoRenewalAuditDetailUsesCertificateSerialNumber(t *testing.T) {
	serialNumber := "01:9F:99:C5:E0:D6:C7:99:D8:28"
	if detail := autoRenewalAuditDetail(serialNumber); detail != "原证书 "+serialNumber+" 已自动续期并删除" {
		t.Fatalf("autoRenewalAuditDetail() = %q", detail)
	}
}
