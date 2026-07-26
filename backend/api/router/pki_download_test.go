package router

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http/httptest"
	"testing"
)

func TestParseCRLDownloadFilterRetainsTableFilters(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/crl/download?caId=ca-1&reason=keyCompromise&keyword=api.example.com", nil)
	filter, err := parseCRLFilter(request)
	if err != nil {
		t.Fatalf("parseCRLFilter() error = %v", err)
	}
	if filter.CAID != "ca-1" {
		t.Fatalf("CAID = %q, want ca-1", filter.CAID)
	}
	if filter.Reason != "keyCompromise" || filter.Keyword != "api.example.com" {
		t.Fatalf("download filter = %#v, want table filters", filter)
	}
}

func TestCRLDownloadArchiveContainsOneFilePerCA(t *testing.T) {
	archive, err := crlDownloadArchive([]string{"ca-one", "ca-two"}, [][]byte{{1, 2, 3}, {4, 5, 6}}, "2026-07-26")
	if err != nil {
		t.Fatalf("crlDownloadArchive() error = %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("archive files = %d, want 2", len(reader.File))
	}
	if reader.File[0].Name != "pki-crl-ca-ca-one-2026-07-26.crl" || reader.File[1].Name != "pki-crl-ca-ca-two-2026-07-26.crl" {
		t.Fatalf("archive filenames = %q, %q", reader.File[0].Name, reader.File[1].Name)
	}
}

func TestCertificateArchiveContainsBundleAndPrivateKey(t *testing.T) {
	archive, err := certificateArchive("api.example.com", "leaf-certificate", "issuer-certificate", "private-key")
	if err != nil {
		t.Fatalf("certificateArchive() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	contents := map[string]string{}
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(entry)
		_ = entry.Close()
		if err != nil {
			t.Fatalf("read %s: %v", file.Name, err)
		}
		contents[file.Name] = string(body)
	}
	if got := contents["api.example.com_bundle.crt"]; got != "leaf-certificate\nissuer-certificate" {
		t.Fatalf("bundle content = %q", got)
	}
	if got := contents["api.example.com.key"]; got != "private-key" {
		t.Fatalf("private key content = %q", got)
	}
}

func TestCSRArchiveContainsRequestAndPrivateKey(t *testing.T) {
	archive, err := csrArchive("api.example.com", "certificate-request", "private-key")
	if err != nil {
		t.Fatalf("csrArchive() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	contents := map[string]string{}
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(entry)
		_ = entry.Close()
		if err != nil {
			t.Fatalf("read %s: %v", file.Name, err)
		}
		contents[file.Name] = string(body)
	}
	if got := contents["api.example.com.csr"]; got != "certificate-request" {
		t.Fatalf("CSR content = %q", got)
	}
	if got := contents["api.example.com.key"]; got != "private-key" {
		t.Fatalf("private key content = %q", got)
	}
}
