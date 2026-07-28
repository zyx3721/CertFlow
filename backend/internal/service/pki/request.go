package pki

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"certflow/backend/internal/domain"

	"github.com/google/uuid"
)

func (s *Service) CreateRequest(ctx context.Context, in RequestInput, actor domain.User, ip string) (domain.Certificate, error) {
	return s.createRequest(ctx, in, actor, ip, true)
}

func (s *Service) createRequest(ctx context.Context, in RequestInput, actor domain.User, ip string, notify bool) (domain.Certificate, error) {
	subject, err := parseSubject(in.Subject)
	if err != nil {
		return domain.Certificate{}, err
	}
	if err := validateRequestInput(&in, subject); err != nil {
		return domain.Certificate{}, err
	}
	var notAfter time.Time
	err = s.store.Pool.QueryRow(ctx, "SELECT not_after FROM certificate_authorities WHERE id=$1 AND status='active'", in.CAID).Scan(&notAfter)
	if err != nil {
		return domain.Certificate{}, errors.New("请选择有效的 CA")
	}
	requestedNotAfter := time.Now().AddDate(0, 0, in.ValidityDays)
	if requestedNotAfter.After(notAfter) {
		return domain.Certificate{}, errors.New("证书有效期不能超过签发 CA 有效期")
	}
	if err := validateCSRKeyPair(in.CSRPEM, in.PrivateKeyPEM); err != nil {
		return domain.Certificate{}, err
	}
	privateKeyCiphertext := ""
	if in.PrivateKeyPEM != "" {
		privateKeyCiphertext, err = s.box.Seal(in.PrivateKeyPEM)
		if err != nil {
			return domain.Certificate{}, err
		}
	}
	san, _ := json.Marshal(in.SAN)
	certificateID := uuid.NewString()
	tx, err := s.store.Pool.Begin(ctx)
	if err != nil {
		return domain.Certificate{}, err
	}
	defer tx.Rollback(ctx)

	requestSerial := pendingSerial()
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO certificates(
			id, ca_id, serial_number, common_name, subject, san, algorithm, purpose,
			applicant_id, applicant, csr_pem, private_key_ciphertext, validity_days, not_after
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id::text
	`, certificateID, in.CAID, requestSerial, in.CommonName, in.Subject, san, in.Algorithm, in.Purpose, actor.ID, actor.Username, in.CSRPEM, privateKeyCiphertext, in.ValidityDays, requestedNotAfter).Scan(&id)
	if err != nil {
		return domain.Certificate{}, err
	}
	workflowID := uuid.NewString()
	_, err = tx.Exec(ctx, "INSERT INTO workflows(id,certificate_id) VALUES($1,$2)", workflowID, id)
	if err != nil {
		return domain.Certificate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Certificate{}, err
	}
	auditActor, auditIP := actor, ip
	if !notify {
		auditActor, auditIP = domain.User{}, ""
	}
	s.store.Audit(ctx, auditActor, "提交证书申请", in.CommonName, "certificates", "success", auditIP, "证书申请 "+workflowID+" 已提交")
	if notify && s.notifier != nil && shouldNotifyApproval(actor) {
		go s.notifier.NotifyApproval(context.Background(), actor.Username, in.CommonName, time.Now())
	}
	return domain.Certificate{ID: id, CAID: in.CAID, SerialNumber: requestSerial, CommonName: in.CommonName, Subject: in.Subject, Algorithm: in.Algorithm, Purpose: in.Purpose, Status: "pending", Applicant: actor.Username, SAN: in.SAN, NotAfter: requestedNotAfter}, nil
}

func shouldNotifyApproval(actor domain.User) bool {
	for _, permission := range actor.Permissions {
		if permission == domain.PermissionWorkflowApprove {
			return false
		}
	}
	return true
}
