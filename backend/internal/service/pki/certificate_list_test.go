package pki

import (
	"strings"
	"testing"
)

func TestCertificateSource(t *testing.T) {
	if got := certificateSource(""); got != "csr" {
		t.Fatalf("empty encrypted key source = %q, want csr", got)
	}
	if got := certificateSource("encrypted-private-key"); got != "system" {
		t.Fatalf("encrypted key source = %q, want system", got)
	}
}

func TestCertificatePEMFingerprint(t *testing.T) {
	if got := certificatePEMFingerprint("not a certificate"); got != "" {
		t.Fatalf("invalid PEM fingerprint = %q, want empty", got)
	}
}

func TestFingerprintUsesColonSeparatedSHA256(t *testing.T) {
	got := fingerprint([]byte("certificate"))
	if strings.Count(got, ":") != 31 {
		t.Fatalf("fingerprint separator count = %d, want 31: %q", strings.Count(got, ":"), got)
	}
	if got != strings.ToUpper(got) {
		t.Fatalf("fingerprint must use uppercase hexadecimal: %q", got)
	}
}

func TestCertificateRequestFingerprintMatchesPKICertFormat(t *testing.T) {
	got := certificateRequestFingerprint("example.com", "REQ:12345678", "RSA-2048")
	const want = "65:89:83:A0:B4:C1:CB:A5:EB:08:17:0D:11:2E:28:30"
	if got != want {
		t.Fatalf("request fingerprint = %q, want %q", got, want)
	}
}
