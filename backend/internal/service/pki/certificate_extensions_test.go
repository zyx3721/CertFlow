package pki

import (
	"crypto/x509"
	"testing"
)

func TestApplyCertificateDistributionEndpoints(t *testing.T) {
	template := &x509.Certificate{}
	applyCertificateDistributionEndpoints(template, map[string]any{
		"crlEnabled":  true,
		"crlUrl":      "https://pki.example.com/crl/",
		"ocspEnabled": true,
		"ocspUrl":     "https://pki.example.com/ocsp",
	}, "issuer-ca")

	if got, want := template.CRLDistributionPoints, []string{"https://pki.example.com/crl/issuer-ca.crl"}; !sameStrings(got, want) {
		t.Fatalf("CRL distribution points = %v, want %v", got, want)
	}
	if got, want := template.OCSPServer, []string{"https://pki.example.com/ocsp"}; !sameStrings(got, want) {
		t.Fatalf("OCSP servers = %v, want %v", got, want)
	}
}

func TestApplyCertificateDistributionEndpointsSkipsDisabledServices(t *testing.T) {
	template := &x509.Certificate{}
	applyCertificateDistributionEndpoints(template, map[string]any{
		"crlEnabled":  false,
		"crlUrl":      "https://pki.example.com/crl",
		"ocspEnabled": false,
		"ocspUrl":     "https://pki.example.com/ocsp",
	}, "issuer-ca")
	if len(template.CRLDistributionPoints) != 0 || len(template.OCSPServer) != 0 {
		t.Fatal("disabled services must not be embedded in certificate extensions")
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
