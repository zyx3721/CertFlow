package pki

import (
	"testing"
	"time"

	"golang.org/x/crypto/ocsp"
)

func TestNormalizeOCSPSerial(t *testing.T) {
	if got := normalizeOCSPSerial("01:af-2B"); got != "01AF2B" {
		t.Fatalf("normalizeOCSPSerial() = %q, want %q", got, "01AF2B")
	}
	if got := normalizeOCSPSerial("xyz"); got != "" {
		t.Fatalf("normalizeOCSPSerial() = %q, want empty", got)
	}
	if got := normalizeOCSPSerial("REQ:00395506"); got != "REQ:00395506" {
		t.Fatalf("normalizeOCSPSerial() = %q, want request serial", got)
	}
}

func TestOCSPCertificateValidityAndTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 26, 1, 2, 3, 0, time.Local)
	notBefore := now.Add(-time.Hour)
	notAfter := now.Add(time.Hour)
	item := ocspCertificateStatus{Status: "valid", NotBefore: &notBefore, NotAfter: &notAfter}
	if !isOCSPCertificateValid(item, now) {
		t.Fatal("certificate in its validity window must be good")
	}
	if got := ocspTimestamp(&notBefore); got != notBefore.Format("2006-01-02 15:04:05") {
		t.Fatalf("ocspTimestamp() = %q", got)
	}
	if isOCSPCertificateValid(item, notAfter.Add(time.Second)) {
		t.Fatal("expired certificate must not be good")
	}
}

func TestOCSPReasonCode(t *testing.T) {
	if got := ocspReasonCode("keyCompromise"); got != ocsp.KeyCompromise {
		t.Fatalf("ocspReasonCode() = %d, want %d", got, ocsp.KeyCompromise)
	}
	if got := ocspReasonCode("unknown"); got != ocsp.Unspecified {
		t.Fatalf("unknown reason code = %d, want unspecified", got)
	}
}
