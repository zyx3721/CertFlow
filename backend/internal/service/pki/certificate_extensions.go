package pki

import (
	"crypto/x509"
	"strings"
)

func applyCertificateDistributionEndpoints(template *x509.Certificate, settings map[string]any, caID string) {
	if settingEnabled(settings, "crlEnabled") {
		if endpoint := strings.TrimRight(settingURL(settings, "crlUrl"), "/"); endpoint != "" {
			template.CRLDistributionPoints = []string{endpoint + "/" + caID + ".crl"}
		}
	}
	if settingEnabled(settings, "ocspEnabled") {
		if endpoint := settingURL(settings, "ocspUrl"); endpoint != "" {
			template.OCSPServer = []string{endpoint}
		}
	}
}

func settingEnabled(settings map[string]any, key string) bool {
	value, ok := settings[key].(bool)
	return ok && value
}

func settingURL(settings map[string]any, key string) string {
	value, _ := settings[key].(string)
	return strings.TrimSpace(value)
}
