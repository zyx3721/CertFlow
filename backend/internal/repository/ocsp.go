package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type OCSPResponder struct {
	ID                   string
	IssuerCAID           string
	CertificatePEM       string
	PrivateKeyCiphertext string
	NotBefore            time.Time
	NotAfter             time.Time
}

func (s *Store) OCSPResponder(ctx context.Context, issuerCAID string) (OCSPResponder, error) {
	var item OCSPResponder
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text,issuer_ca_id::text,certificate_pem,private_key_ciphertext,not_before,not_after
		FROM ocsp_responders WHERE issuer_ca_id=$1
	`, issuerCAID).Scan(&item.ID, &item.IssuerCAID, &item.CertificatePEM, &item.PrivateKeyCiphertext, &item.NotBefore, &item.NotAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}

func (s *Store) CreateOCSPResponder(ctx context.Context, item OCSPResponder) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO ocsp_responders(id,issuer_ca_id,certificate_pem,private_key_ciphertext,not_before,not_after)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(issuer_ca_id) DO NOTHING
	`, item.ID, item.IssuerCAID, item.CertificatePEM, item.PrivateKeyCiphertext, item.NotBefore, item.NotAfter)
	return err
}
