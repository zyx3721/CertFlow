package pki

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"certflow/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ocsp"
)

type OCSPStatus struct {
	Status       string  `json:"status"`
	Valid        bool    `json:"valid"`
	Reason       *string `json:"reason"`
	SerialNumber string  `json:"serialNumber"`
	CommonName   string  `json:"commonName,omitempty"`
	RevokedAt    string  `json:"revokedAt,omitempty"`
	NotBefore    string  `json:"notBefore,omitempty"`
	NotAfter     string  `json:"notAfter,omitempty"`
	CheckedAt    string  `json:"checkedAt"`
}

type ocspCertificateStatus struct {
	Status       string
	Reason       string
	RevokedAt    *time.Time
	IssuerCAID   string
	SerialNumber string
	CommonName   string
	NotBefore    *time.Time
	NotAfter     *time.Time
}

// EnsureOCSPResponders creates the delegated OCSP signing identity for every
// active issuing CA. It is safe to call during startup and after migrations.
func (s *Service) EnsureOCSPResponders(ctx context.Context) error {
	rows, err := s.store.Pool.Query(ctx, "SELECT id::text FROM certificate_authorities WHERE type='issuing' AND status='active'")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var issuerCAID string
		if err := rows.Scan(&issuerCAID); err != nil {
			return err
		}
		if _, _, _, err := s.ensureOCSPResponder(ctx, issuerCAID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) OCSPStatus(ctx context.Context, serialNumber string) (OCSPStatus, error) {
	serial := normalizeOCSPSerial(serialNumber)
	result := OCSPStatus{Status: "unknown", Valid: false, SerialNumber: displayCertificateSerial(serial), CheckedAt: time.Now().Local().Format("2006-01-02 15:04:05")}
	if serial == "" {
		result.Reason = ocspStatusReason("missingCertificate")
		return result, nil
	}
	item, err := s.ocspCertificateStatus(ctx, serial)
	if errors.Is(err, repository.ErrNotFound) {
		result.Reason = ocspStatusReason("notFound")
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.SerialNumber, result.CommonName = displayCertificateSerial(item.SerialNumber), item.CommonName
	if item.Status == "revoked" {
		result.Status, result.Reason = "revoked", ocspStatusReason(item.Reason)
		if item.RevokedAt != nil {
			result.RevokedAt = item.RevokedAt.Local().Format("2006-01-02 15:04:05")
		}
		return result, nil
	}
	if !isOCSPCertificateValid(item, time.Now()) {
		result.Reason = ocspStatusReason("notValid")
		result.NotBefore = ocspTimestamp(item.NotBefore)
		result.NotAfter = ocspTimestamp(item.NotAfter)
		return result, nil
	}
	result.Status, result.Valid = "good", true
	result.NotBefore = ocspTimestamp(item.NotBefore)
	result.NotAfter = ocspTimestamp(item.NotAfter)
	return result, nil
}

func (s *Service) OCSP(ctx context.Context, requestBytes []byte) ([]byte, error) {
	request, err := ocsp.ParseRequest(requestBytes)
	if err != nil {
		return ocsp.MalformedRequestErrorResponse, nil
	}
	serial := strings.ToUpper(hex.EncodeToString(request.SerialNumber.Bytes()))
	item, err := s.ocspCertificateStatus(ctx, serial)
	if errors.Is(err, repository.ErrNotFound) {
		return ocsp.UnauthorizedErrorResponse, nil
	}
	if err != nil {
		return nil, err
	}
	issuer, responder, signer, err := s.ensureOCSPResponder(ctx, item.IssuerCAID)
	if err != nil {
		return nil, err
	}
	response := ocsp.Response{SerialNumber: request.SerialNumber, Status: ocsp.Good, ThisUpdate: time.Now(), NextUpdate: time.Now().Add(time.Hour)}
	if item.Status == "revoked" {
		response.Status = ocsp.Revoked
		response.RevokedAt = *item.RevokedAt
		response.RevocationReason = ocspReasonCode(item.Reason)
	} else if !isOCSPCertificateValid(item, time.Now()) {
		response.Status = ocsp.Unknown
	}
	return ocsp.CreateResponse(issuer, responder, response, signer)
}

func (s *Service) ocspCertificateStatus(ctx context.Context, serial string) (ocspCertificateStatus, error) {
	var item ocspCertificateStatus
	err := s.store.Pool.QueryRow(ctx, `
		SELECT c.status,c.revoke_reason,c.revoked_at,c.ca_id::text,c.serial_number,c.common_name,COALESCE(c.not_before,c.created_at),c.not_after
		FROM certificates c WHERE c.serial_number=$1
	`, serial).Scan(&item.Status, &item.Reason, &item.RevokedAt, &item.IssuerCAID, &item.SerialNumber, &item.CommonName, &item.NotBefore, &item.NotAfter)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return item, err
	}
	err = s.store.Pool.QueryRow(ctx, `
		SELECT 'revoked',r.revoke_reason,r.revoked_at,r.ca_id::text,r.serial_number,r.common_name,NULL::timestamptz,NULL::timestamptz
		FROM certificate_revocation_entries r WHERE r.serial_number=$1
	`, serial).Scan(&item.Status, &item.Reason, &item.RevokedAt, &item.IssuerCAID, &item.SerialNumber, &item.CommonName, &item.NotBefore, &item.NotAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, repository.ErrNotFound
	}
	if err != nil {
		return item, err
	}
	return item, nil
}

func (s *Service) ensureOCSPResponder(ctx context.Context, issuerCAID string) (*x509.Certificate, *x509.Certificate, crypto.Signer, error) {
	issuer, issuerKey, algorithm, name, notAfter, err := s.ocspIssuer(ctx, issuerCAID)
	if err != nil {
		return nil, nil, nil, err
	}
	item, err := s.store.OCSPResponder(ctx, issuerCAID)
	if errors.Is(err, repository.ErrNotFound) {
		item, err = s.createOCSPResponder(ctx, issuerCAID, issuer, issuerKey, algorithm, name, notAfter)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	block, _ := pem.Decode([]byte(item.CertificatePEM))
	if block == nil {
		return nil, nil, nil, errors.New("invalid OCSP responder certificate")
	}
	responder, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, nil, err
	}
	plain, err := s.box.Open(item.PrivateKeyCiphertext)
	if err != nil {
		return nil, nil, nil, err
	}
	block, _ = pem.Decode([]byte(plain))
	if block == nil {
		return nil, nil, nil, errors.New("invalid OCSP responder private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, nil, errors.New("invalid OCSP responder signer")
	}
	return issuer, responder, signer, nil
}

func (s *Service) ocspIssuer(ctx context.Context, issuerCAID string) (*x509.Certificate, crypto.Signer, string, string, time.Time, error) {
	var certificatePEM, ciphertext, algorithm, name string
	var notAfter time.Time
	err := s.store.Pool.QueryRow(ctx, `
		SELECT certificate_pem,private_key_ciphertext,algorithm,name,not_after
		FROM certificate_authorities WHERE id=$1 AND status='active'
	`, issuerCAID).Scan(&certificatePEM, &ciphertext, &algorithm, &name, &notAfter)
	if err != nil {
		return nil, nil, "", "", time.Time{}, repository.ErrNotFound
	}
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil {
		return nil, nil, "", "", time.Time{}, errors.New("invalid issuer certificate")
	}
	issuer, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, "", "", time.Time{}, err
	}
	plain, err := s.box.Open(ciphertext)
	if err != nil {
		return nil, nil, "", "", time.Time{}, err
	}
	block, _ = pem.Decode([]byte(plain))
	if block == nil {
		return nil, nil, "", "", time.Time{}, errors.New("invalid issuer private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, "", "", time.Time{}, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, "", "", time.Time{}, errors.New("invalid issuer signer")
	}
	return issuer, signer, algorithm, name, notAfter, nil
}

func (s *Service) createOCSPResponder(ctx context.Context, issuerCAID string, issuer *x509.Certificate, issuerKey crypto.Signer, algorithm, issuerName string, notAfter time.Time) (repository.OCSPResponder, error) {
	key, err := keyFor(algorithm)
	if err != nil {
		return repository.OCSPResponder{}, err
	}
	serialText, serial, err := randomSerial()
	if err != nil {
		return repository.OCSPResponder{}, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "OCSP Responder for " + issuerName},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
		BasicConstraintsValid: true,
		AuthorityKeyId:        issuer.SubjectKeyId,
		SubjectKeyId:          subjectKeyID([]byte("ocsp-responder-" + issuerCAID + serialText)),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, key.Public(), issuerKey)
	if err != nil {
		return repository.OCSPResponder{}, err
	}
	keyPEM, err := pemKey(key)
	if err != nil {
		return repository.OCSPResponder{}, err
	}
	ciphertext, err := s.box.Seal(keyPEM)
	if err != nil {
		return repository.OCSPResponder{}, err
	}
	item := repository.OCSPResponder{ID: uuid.NewString(), IssuerCAID: issuerCAID, CertificatePEM: pemCert(der), PrivateKeyCiphertext: ciphertext, NotBefore: template.NotBefore, NotAfter: template.NotAfter}
	if err := s.store.CreateOCSPResponder(ctx, item); err != nil {
		return item, err
	}
	return s.store.OCSPResponder(ctx, issuerCAID)
}

func normalizeOCSPSerial(value string) string {
	candidate := strings.ToUpper(strings.TrimSpace(value))
	if strings.HasPrefix(candidate, "REQ:") {
		requestNumber := strings.TrimPrefix(candidate, "REQ:")
		if requestNumber != "" {
			valid := true
			for _, character := range requestNumber {
				if character < '0' || character > '9' {
					valid = false
					break
				}
			}
			if valid {
				return candidate
			}
		}
	}
	var builder strings.Builder
	for _, character := range value {
		if ('0' <= character && character <= '9') || ('a' <= character && character <= 'f') || ('A' <= character && character <= 'F') {
			builder.WriteRune(character)
		}
	}
	return strings.ToUpper(builder.String())
}

func ocspReasonCode(reason string) int {
	return map[string]int{"keyCompromise": ocsp.KeyCompromise, "cACompromise": ocsp.CACompromise, "affiliationChanged": ocsp.AffiliationChanged, "superseded": ocsp.Superseded, "cessationOfOperation": ocsp.CessationOfOperation}[reason]
}

func ocspStatusReason(value string) *string {
	return &value
}

func ocspTimestamp(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func isOCSPCertificateValid(item ocspCertificateStatus, now time.Time) bool {
	return item.Status == "valid" && item.NotBefore != nil && item.NotAfter != nil && !now.Before(*item.NotBefore) && !now.After(*item.NotAfter)
}
