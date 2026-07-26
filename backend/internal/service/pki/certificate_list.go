package pki

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"strings"
	"time"

	"certflow/backend/internal/domain"
)

func (s *Service) ListCertificates(ctx context.Context) ([]domain.Certificate, error) {
	query := `SELECT id::text,ca_id::text,serial_number,common_name,subject,algorithm,purpose,status,applicant,san,COALESCE(not_before,now()),COALESCE(not_after,now()),COALESCE(revoked_at,now()),revoke_reason,reject_reason,certificate_pem,private_key_ciphertext FROM certificates`
	query += " ORDER BY created_at DESC"
	rows, err := s.store.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Certificate{}
	for rows.Next() {
		var item domain.Certificate
		var san []byte
		var revoked time.Time
		var privateKeyCiphertext string
		if err := rows.Scan(&item.ID, &item.CAID, &item.SerialNumber, &item.CommonName, &item.Subject, &item.Algorithm, &item.Purpose, &item.Status, &item.Applicant, &san, &item.NotBefore, &item.NotAfter, &revoked, &item.RevokeReason, &item.RejectReason, &item.CertificatePEM, &privateKeyCiphertext); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(san, &item.SAN)
		item.SerialNumber = displayCertificateSerial(item.SerialNumber)
		item.Source = certificateSource(privateKeyCiphertext)
		item.Fingerprint = certificateFingerprint(item.CertificatePEM, item.CommonName, item.SerialNumber, item.Algorithm)
		if item.Status == "revoked" {
			item.RevokedAt = &revoked
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func certificateSource(privateKeyCiphertext string) string {
	if strings.TrimSpace(privateKeyCiphertext) == "" {
		return "csr"
	}
	return "system"
}

func certificatePEMFingerprint(certificatePEM string) string {
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil {
		return ""
	}
	return fingerprint(block.Bytes)
}

func certificateFingerprint(certificatePEM, commonName, serialNumber, algorithm string) string {
	if fingerprint := certificatePEMFingerprint(certificatePEM); fingerprint != "" {
		return fingerprint
	}
	return certificateRequestFingerprint(commonName, serialNumber, algorithm)
}
