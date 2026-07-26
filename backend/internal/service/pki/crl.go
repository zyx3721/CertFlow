package pki

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"certflow/backend/internal/repository"
)

func (s *Service) CRLWithInterval(ctx context.Context, caID string, intervalHours int) ([]byte, error) {
	return s.CRLWithFilter(ctx, caID, intervalHours, CRLFilter{})
}

func (s *Service) CRLWithFilter(ctx context.Context, caID string, intervalHours int, filter CRLFilter) ([]byte, error) {
	return s.buildCRL(ctx, caID, intervalHours, time.Now().UnixNano(), filter)
}

func (s *Service) buildCRL(ctx context.Context, caID string, intervalHours int, number int64, filter CRLFilter) ([]byte, error) {
	if intervalHours < 1 {
		intervalHours = 24
	}
	var certificatePEM, cipher string
	if err := s.store.Pool.QueryRow(ctx, "SELECT certificate_pem,private_key_ciphertext FROM certificate_authorities WHERE id=$1 AND status='active'", caID).Scan(&certificatePEM, &cipher); err != nil {
		return nil, repository.ErrNotFound
	}
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil {
		return nil, errors.New("invalid CA certificate")
	}
	issuer, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	plain, err := s.box.Open(cipher)
	if err != nil {
		return nil, err
	}
	block, _ = pem.Decode([]byte(plain))
	if block == nil {
		return nil, errors.New("invalid CA private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	query, args := crlCertificateQuery(caID, filter)
	rows, err := s.store.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	revoked := []x509.RevocationListEntry{}
	for rows.Next() {
		var serialText, reason string
		var at time.Time
		if err := rows.Scan(&serialText, &at, &reason); err != nil {
			return nil, err
		}
		number := new(big.Int)
		if _, ok := number.SetString(serialText, 16); !ok {
			return nil, errors.New("invalid certificate serial number")
		}
		revoked = append(revoked, x509.RevocationListEntry{SerialNumber: number, RevocationTime: at, ReasonCode: crlReasonCode(reason)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:                    big.NewInt(number),
		ThisUpdate:                now,
		NextUpdate:                now.Add(time.Duration(intervalHours) * time.Hour),
		RevokedCertificateEntries: revoked,
	}, issuer, key.(crypto.Signer))
}

func crlCertificateQuery(caID string, filter CRLFilter) (string, []any) {
	filter.CAID = caID
	where, args := crlWhere(filter)
	return `SELECT r.serial_number,r.revoked_at,r.revoke_reason
		FROM certificate_revocation_entries r
		JOIN certificate_authorities ca ON ca.id=r.ca_id
		WHERE true` + where + `
		ORDER BY r.revoked_at DESC`, args
}

func crlReasonCode(reason string) int {
	return map[string]int{
		"keyCompromise":        1,
		"cACompromise":         2,
		"affiliationChanged":   3,
		"superseded":           4,
		"cessationOfOperation": 5,
	}[reason]
}
