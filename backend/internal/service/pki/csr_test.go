package pki

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateCSRBuildsMatchingKeyMaterial(t *testing.T) {
	service := New(nil, nil)
	preview, err := service.GenerateCSR(context.Background(), RequestInput{
		Subject:   "CN=api.acme.com, O=Acme Corp, C=CN",
		Algorithm: "RSA-2048",
		SAN:       []string{"api.acme.com", "10.0.0.10"},
	})
	if err != nil {
		t.Fatalf("GenerateCSR() error = %v", err)
	}
	if err := validateCSRKeyPair(preview.CSRPEM, preview.PrivateKeyPEM); err != nil {
		t.Fatalf("generated key material failed validation: %v", err)
	}
	block, _ := pem.Decode([]byte(preview.CSRPEM))
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificateRequest() error = %v", err)
	}
	if csr.Subject.CommonName != "api.acme.com" || len(csr.DNSNames) != 1 || len(csr.IPAddresses) != 1 {
		t.Fatalf("unexpected CSR content: subject=%q dns=%v ips=%v", csr.Subject.CommonName, csr.DNSNames, csr.IPAddresses)
	}
}

func TestInspectCSRExtractsCertificateRequestFields(t *testing.T) {
	service := New(nil, nil)
	preview, err := service.GenerateCSR(context.Background(), RequestInput{
		Subject:   "CN=api.acme.com, O=Acme Corp, OU=Security, C=CN, ST=Shanghai, L=Shanghai",
		Algorithm: "ECDSA-P256",
		SAN:       []string{"api.acme.com", "10.0.0.10"},
	})
	if err != nil {
		t.Fatalf("GenerateCSR() error = %v", err)
	}
	inspection, err := service.InspectCSR(context.Background(), preview.CSRPEM)
	if err != nil {
		t.Fatalf("InspectCSR() error = %v", err)
	}
	if inspection.Subject.CommonName != "api.acme.com" || inspection.Subject.Org != "Acme Corp" {
		t.Fatalf("unexpected subject: %+v", inspection.Subject)
	}
	if inspection.Algorithm != "ECDSA-P256" || len(inspection.SAN) != 2 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}
