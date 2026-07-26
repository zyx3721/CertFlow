package pki

import (
	"crypto/x509/pkix"
	"regexp"
	"strings"
	"testing"
)

func TestValidateRequestInput(t *testing.T) {
	base := func() RequestInput {
		return RequestInput{
			CommonName:   "api.acme.com",
			Algorithm:    "RSA-2048",
			Purpose:      "server",
			SAN:          []string{" api.acme.com ", "10.0.0.10"},
			ValidityDays: 365,
		}
	}
	subject := pkix.Name{CommonName: "api.acme.com", Country: []string{"CN"}}

	t.Run("accepts and normalizes a valid request", func(t *testing.T) {
		input := base()
		input.ValidityDays = 7300
		if err := validateRequestInput(&input, subject); err != nil {
			t.Fatalf("validateRequestInput() error = %v", err)
		}
		if input.SAN[0] != "api.acme.com" {
			t.Fatalf("SAN was not normalized: %q", input.SAN[0])
		}
	})

	t.Run("rejects a validity period beyond 20 years", func(t *testing.T) {
		input := base()
		input.ValidityDays = 7301
		if err := validateRequestInput(&input, subject); err == nil {
			t.Fatal("validateRequestInput() error = nil")
		}
	})

	t.Run("accepts mutual TLS", func(t *testing.T) {
		input := base()
		input.Purpose = "mtls"
		if err := validateRequestInput(&input, subject); err != nil {
			t.Fatalf("validateRequestInput() error = %v", err)
		}
	})

	for name, mutate := range map[string]func(*RequestInput, *pkix.Name){
		"rejects an empty SAN": func(input *RequestInput, _ *pkix.Name) { input.SAN = nil },
		"rejects an invalid IPv4 address": func(input *RequestInput, _ *pkix.Name) {
			input.SAN = []string{"999.0.0.1"}
		},
		"rejects a mismatched common name": func(input *RequestInput, _ *pkix.Name) {
			input.CommonName = "other.acme.com"
		},
		"rejects an invalid country code": func(_ *RequestInput, name *pkix.Name) {
			name.Country = []string{"CHN"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := base()
			name := subject
			mutate(&input, &name)
			if err := validateRequestInput(&input, name); err == nil {
				t.Fatal("validateRequestInput() error = nil")
			}
		})
	}
}

func TestValidateRequestInputSupportsAllClientAlgorithms(t *testing.T) {
	for _, algorithm := range []string{"RSA-2048", "RSA-4096", "ECDSA-P256", "ECDSA-P384", "ED25519"} {
		t.Run(algorithm, func(t *testing.T) {
			input := RequestInput{
				CommonName:   "api.acme.com",
				Algorithm:    algorithm,
				Purpose:      "server",
				SAN:          []string{"api.acme.com"},
				ValidityDays: 365,
			}
			err := validateRequestInput(&input, pkix.Name{CommonName: "api.acme.com"})
			if err != nil {
				t.Fatalf("validateRequestInput() error = %v", err)
			}
		})
	}

	input := RequestInput{CommonName: "api.acme.com", Algorithm: "RSA-1024", Purpose: "server", SAN: []string{"api.acme.com"}, ValidityDays: 365}
	err := validateRequestInput(&input, pkix.Name{CommonName: "api.acme.com"})
	if err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("unsupported algorithm error = %v", err)
	}
}

func TestPendingSerial(t *testing.T) {
	first := pendingSerial()
	second := pendingSerial()
	if !regexp.MustCompile(`^REQ:\d{8}$`).MatchString(first) {
		t.Fatalf("pending serial prefix = %q", first)
	}
	if first == second {
		t.Fatalf("pending serial must be unique: %q", first)
	}
}

func TestIssuedSerialFormat(t *testing.T) {
	serialText, _, err := randomSerial()
	if err != nil {
		t.Fatalf("randomSerial() error = %v", err)
	}
	if !regexp.MustCompile(`^[A-F0-9]{20}$`).MatchString(serialText) {
		t.Fatalf("stored serial format = %q", serialText)
	}
	if displayed := displayIssuedSerial(serialText); !regexp.MustCompile(`^(?:[A-F0-9]{2}:){9}[A-F0-9]{2}$`).MatchString(displayed) {
		t.Fatalf("displayed serial format = %q", displayed)
	}
}
