package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type AutoRenewalCandidate struct {
	ID           string
	SerialNumber string
	CAID         string
	CommonName   string
	Subject      string
	SAN          []string
	Algorithm    string
	Purpose      string
	ValidityDays int
	ApplicantID  string
	Applicant    string
}

func (s *Store) ListAutoRenewalCandidates(ctx context.Context, until time.Time) ([]AutoRenewalCandidate, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.id::text,c.serial_number,c.ca_id::text,c.common_name,c.subject,c.san,c.algorithm,c.purpose,c.validity_days,
		       COALESCE(c.applicant_id::text,''),c.applicant
		FROM certificates c
		WHERE c.status='valid' AND c.not_after>now() AND c.not_after<=$1
		  AND NOT EXISTS (SELECT 1 FROM certificates renewal WHERE renewal.renewed_from_id=c.id)
		ORDER BY c.not_after,c.common_name
	`, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AutoRenewalCandidate{}
	for rows.Next() {
		var item AutoRenewalCandidate
		var sanRaw []byte
		if err := rows.Scan(&item.ID, &item.SerialNumber, &item.CAID, &item.CommonName, &item.Subject, &sanRaw, &item.Algorithm, &item.Purpose, &item.ValidityDays, &item.ApplicantID, &item.Applicant); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(sanRaw, &item.SAN); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) LinkAutoRenewal(ctx context.Context, renewalID, originalID string) error {
	_, err := s.Pool.Exec(ctx, "UPDATE certificates SET renewed_from_id=$2,updated_at=now() WHERE id=$1", renewalID, originalID)
	return err
}

func (s *Store) WorkflowIDForCertificate(ctx context.Context, certificateID string) (string, error) {
	var workflowID string
	err := s.Pool.QueryRow(ctx, "SELECT id::text FROM workflows WHERE certificate_id=$1", certificateID).Scan(&workflowID)
	return workflowID, err
}

func (s *Store) DeleteCertificateRequest(ctx context.Context, certificateID string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM certificates WHERE id=$1 AND status='pending'", certificateID)
	return err
}

func (s *Store) CertificateNotAfter(ctx context.Context, certificateID string) (time.Time, error) {
	var notAfter time.Time
	err := s.Pool.QueryRow(ctx, "SELECT not_after FROM certificates WHERE id=$1 AND status='valid'", certificateID).Scan(&notAfter)
	return notAfter, err
}

// DeleteRenewedOriginalCertificate removes the source certificate after its
// replacement has been successfully issued by the automatic renewal flow.
// This intentionally differs from the user-facing delete operation, which
// requires valid certificates to be revoked first.
func (s *Store) DeleteRenewedOriginalCertificate(ctx context.Context, originalID string) error {
	result, err := s.Pool.Exec(ctx, "DELETE FROM certificates WHERE id=$1 AND status='valid'", originalID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("original certificate is no longer valid or does not exist")
	}
	return nil
}
