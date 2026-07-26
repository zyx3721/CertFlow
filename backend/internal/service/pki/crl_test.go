package pki

import (
	"strings"
	"testing"
)

func TestCRLCertificateQueryMatchesListFilters(t *testing.T) {
	query, args := crlCertificateQuery("ca-1", CRLFilter{Reason: "affiliationChanged", Keyword: "归属变更"})
	if !strings.Contains(query, "JOIN certificate_authorities ca") {
		t.Fatal("CRL download query must join CA data like the list query")
	}
	if !strings.Contains(query, "WHEN 'affiliationChanged' THEN '归属变更'") {
		t.Fatal("CRL download query must support localized revocation-reason keywords")
	}
	if len(args) != 3 || args[0] != "ca-1" || args[1] != "affiliationChanged" || args[2] != "%归属变更%" {
		t.Fatalf("query args = %#v", args)
	}
}
