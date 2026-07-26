package pki

import "testing"

func TestCertificateOperationAuditDetails(t *testing.T) {
	serialNumber := "01:9F:99:C5:E0:D6:C7:99:D8:28"
	if detail := certificateVerificationAuditDetail(serialNumber, "revoked"); detail != "证书 "+serialNumber+" 校验结果：已撤销" {
		t.Fatalf("verification detail = %q", detail)
	}
	if detail := certificateDownloadAuditDetail(serialNumber); detail != "证书 "+serialNumber+" 已下载" {
		t.Fatalf("download detail = %q", detail)
	}
}
