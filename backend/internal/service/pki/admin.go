package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Service) VerifyCertificate(ctx context.Context, id string, actor domain.User, ip string) (map[string]any, error) {
	var commonName string
	var serialNumber string
	var status string
	var certificatePEM string
	var notAfter *time.Time
	var owner string
	err := s.store.Pool.QueryRow(ctx, `
		SELECT common_name,serial_number,status,certificate_pem,not_after,
		       COALESCE(applicant_id::text,'')
		FROM certificates
		WHERE id=$1
	`, id).Scan(&commonName, &serialNumber, &status, &certificatePEM, &notAfter, &owner)
	if err != nil {
		return nil, repository.ErrNotFound
	}
	if actor.Role != "admin" && owner != actor.ID {
		return nil, errors.New("无权校验该证书")
	}

	valid := status == "valid"
	reason := "certificateValid"
	if status == "revoked" {
		reason = "certificateRevoked"
	} else if status == "expired" || (notAfter != nil && notAfter.Before(time.Now())) {
		valid = false
		status = "expired"
		reason = "certificateExpired"
	} else if status != "valid" {
		reason = "certificateNotIssued"
	}
	if valid {
		block, _ := pem.Decode([]byte(strings.TrimSpace(certificatePEM)))
		if block == nil {
			valid = false
			reason = "certificateMaterialInvalid"
		}
	}
	s.store.Audit(ctx, actor, "校验证书状态", commonName, "certificates", "success", ip, certificateVerificationAuditDetail(serialNumber, status))
	return map[string]any{
		"valid":        valid,
		"status":       status,
		"reason":       reason,
		"checkedAt":    time.Now().UTC(),
		"commonName":   commonName,
		"serialNumber": displayCertificateSerial(serialNumber),
	}, nil
}

func certificateVerificationAuditDetail(serialNumber, status string) string {
	return "证书 " + displayCertificateSerial(serialNumber) + " 校验结果：" + certificateStatusLabel(status)
}

func (s *Service) DeleteCA(ctx context.Context, id string, actor domain.User, ip string) error {
	var name string
	tx, err := s.store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, "SELECT name FROM certificate_authorities WHERE id=$1", id).Scan(&name); err != nil {
		return repository.ErrNotFound
	}

	certificateExists, err := hasRelatedCertificates(ctx, tx, id)
	if err != nil {
		return err
	}
	if certificateExists {
		return errors.New("该 CA 或其下级 CA 已关联证书，不能删除")
	}

	result, err := tx.Exec(ctx, "DELETE FROM certificate_authorities WHERE id=$1", id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return errors.New("CA 层级删除约束尚未更新，请重启后端后重试")
		}
		return errors.New("删除 CA 失败")
	}
	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.store.Audit(ctx, actor, "删除 CA", name, "pki", "success", ip, "CA "+name+" 已删除")
	return nil
}

func (s *Service) CanDeleteCA(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := s.store.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM certificate_authorities WHERE id=$1)", id).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, repository.ErrNotFound
	}
	hasRelated, err := hasRelatedCertificates(ctx, s.store.Pool, id)
	return !hasRelated, err
}

type caDeletionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func hasRelatedCertificates(ctx context.Context, queryer caDeletionQuerier, id string) (bool, error) {
	var exists bool
	err := queryer.QueryRow(ctx, `
		WITH RECURSIVE ca_tree AS (
			SELECT id FROM certificate_authorities WHERE id=$1
			UNION ALL
			SELECT child.id
			FROM certificate_authorities child
			JOIN ca_tree parent ON child.parent_id=parent.id
		)
		SELECT EXISTS(
			SELECT 1 FROM certificates WHERE ca_id IN (SELECT id FROM ca_tree)
			UNION ALL
			SELECT 1 FROM certificate_revocation_entries WHERE ca_id IN (SELECT id FROM ca_tree)
		)
	`, id).Scan(&exists)
	return exists, err
}

func (s *Service) CACertificate(ctx context.Context, id string) (domain.CA, error) {
	var item domain.CA
	err := s.store.Pool.QueryRow(ctx, `
		SELECT id::text,name,type,algorithm,subject,COALESCE(parent_id::text,''),
		       certificate_pem,not_before,not_after,issued_certs,revoked_certs,status
		FROM certificate_authorities
		WHERE id=$1
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.Algorithm,
		&item.Subject,
		&item.ParentID,
		&item.CertificatePEM,
		&item.NotBefore,
		&item.NotAfter,
		&item.IssuedCerts,
		&item.RevokedCerts,
		&item.Status,
	)
	if err != nil {
		return domain.CA{}, repository.ErrNotFound
	}
	return item, nil
}

func (s *Service) DeleteCertificate(ctx context.Context, id string, actor domain.User, ip string) error {
	var name string
	var status, serialNumber string
	if err := s.store.Pool.QueryRow(
		ctx,
		"SELECT common_name,status,serial_number FROM certificates WHERE id=$1",
		id,
	).Scan(&name, &status, &serialNumber); err != nil {
		return repository.ErrNotFound
	}
	if status == "valid" {
		return errors.New("有效证书必须先撤销后才能删除")
	}
	result, err := s.store.Pool.Exec(ctx, "DELETE FROM certificates WHERE id=$1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	s.store.Audit(ctx, actor, "删除证书", name, "certificates", "success", ip, "证书 "+displayCertificateSerial(serialNumber)+" 已删除")
	return nil
}

func (s *Service) DeleteWorkflow(ctx context.Context, id string, actor domain.User, ip string) error {
	var certificateID string
	var certName string
	var status string
	if err := s.store.Pool.QueryRow(ctx, `
		SELECT w.certificate_id::text,c.common_name,w.status
		FROM workflows w
		JOIN certificates c ON c.id=w.certificate_id
		WHERE w.id=$1
	`, id).Scan(&certificateID, &certName, &status); err != nil {
		return repository.ErrNotFound
	}
	if status != "pending" {
		return errors.New("只能删除待审批记录")
	}
	tx, err := s.store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM workflows WHERE id=$1", id); err == nil {
		_, err = tx.Exec(ctx, "DELETE FROM certificates WHERE id=$1 AND status='pending'", certificateID)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.store.Audit(ctx, actor, "删除审批记录", certName, "workflows", "success", ip, "审批单 "+id+" 已删除，关联证书 "+certificateID+" 已删除")
	return nil
}

func (s *Service) CertificateTrend(ctx context.Context, actor domain.User) ([]map[string]any, error) {
	issuedScope := ""
	revokedScope := ""
	args := []any{}
	if actor.Role != "admin" {
		issuedScope = " AND c.applicant_id=$1"
		revokedScope = " AND r.applicant_id=$1"
		args = append(args, actor.ID)
	}

	query := `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', now()) - interval '5 months',
				date_trunc('month', now()),
				interval '1 month'
			) AS month
		)
		SELECT to_char(months.month, 'YYYY-MM'),
		       (SELECT count(*) FROM certificates c
		        WHERE c.status IN ('valid','revoked','expired')
		          AND c.not_before >= months.month
		          AND c.not_before < months.month + interval '1 month'` + issuedScope + `),
		       (SELECT count(*) FROM certificate_revocation_entries r
		        WHERE r.revoked_at >= months.month
		          AND r.revoked_at < months.month + interval '1 month'` + revokedScope + `)
		FROM months
		ORDER BY months.month`

	rows, err := s.store.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var month string
		var issued int
		var revoked int
		if err := rows.Scan(&month, &issued, &revoked); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"month":   month,
			"issued":  issued,
			"revoked": revoked,
		})
	}
	return items, rows.Err()
}

type CRLFilter struct {
	CAID     string
	Reason   string
	Keyword  string
	Page     int
	PageSize int
}

func (s *Service) ListCRLEntries(ctx context.Context, filter CRLFilter) ([]domain.CRLEntry, int, error) {
	where, args := crlWhere(filter)
	var total int
	if err := s.store.Pool.QueryRow(ctx, `
		SELECT count(*)
		FROM certificate_revocation_entries r
		JOIN certificate_authorities ca ON ca.id=r.ca_id
		WHERE true`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 10
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.store.Pool.Query(ctx, `
		SELECT r.id::text,r.serial_number,r.revoked_at,r.revoke_reason,r.ca_id::text,
		       ca.name,r.common_name
		FROM certificate_revocation_entries r
		JOIN certificate_authorities ca ON ca.id=r.ca_id
		WHERE true`+where+`
		ORDER BY r.revoked_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args))+`
	`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.CRLEntry{}
	for rows.Next() {
		var item domain.CRLEntry
		if err := rows.Scan(
			&item.ID,
			&item.SerialNumber,
			&item.RevokedAt,
			&item.Reason,
			&item.CAID,
			&item.CAName,
			&item.CommonName,
		); err != nil {
			return nil, 0, err
		}
		item.SerialNumber = displayIssuedSerial(item.SerialNumber)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Service) CRLMetadata(ctx context.Context, intervalHours int, filter CRLFilter) (map[string]any, error) {
	where, args := crlWhere(filter)
	var count int
	if err := s.store.Pool.QueryRow(ctx, `
		SELECT count(*)
		FROM certificate_revocation_entries r
		JOIN certificate_authorities ca ON ca.id=r.ca_id
		WHERE true`+where, args...).Scan(&count); err != nil {
		return nil, err
	}
	if intervalHours < 1 {
		intervalHours = 24
	}
	thisUpdate := time.Now().UTC()
	return map[string]any{
		"version":            "X.509 v2",
		"signatureAlgorithm": s.crlSignatureAlgorithm(ctx, filter.CAID),
		"thisUpdate":         thisUpdate,
		"nextUpdate":         thisUpdate.Add(time.Duration(intervalHours) * time.Hour),
		"entryCount":         count,
		"intervalHours":      intervalHours,
	}, nil
}

func crlWhere(filter CRLFilter) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 4)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if filter.CAID != "" {
		add(" AND r.ca_id=$%d", filter.CAID)
	}
	if filter.Reason != "" {
		add(" AND r.revoke_reason=$%d", filter.Reason)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		args = append(args, "%"+keyword+"%")
		position := len(args)
		clauses = append(clauses, fmt.Sprintf(` AND (
			r.serial_number ILIKE $%d
			OR r.serial_number ILIKE regexp_replace($%d, ':', '', 'g')
			OR r.common_name ILIKE $%d
			OR to_char(r.revoked_at AT TIME ZONE 'Asia/Shanghai', 'YYYY/MM/DD HH24:MI:SS') ILIKE $%d
			OR r.revoke_reason ILIKE $%d
			OR CASE r.revoke_reason
				WHEN 'unspecified' THEN '未指定'
				WHEN 'keyCompromise' THEN '密钥泄露'
				WHEN 'cACompromise' THEN 'CA 泄露'
				WHEN 'affiliationChanged' THEN '归属变更'
				WHEN 'superseded' THEN '已替换'
				WHEN 'cessationOfOperation' THEN '停止运营'
			END ILIKE $%d
			OR ca.name ILIKE $%d
			OR r.ca_id::text ILIKE $%d
		)`, position, position, position, position, position, position, position, position))
	}
	return strings.Join(clauses, ""), args
}

func (s *Service) ActiveCRLCAIDs(ctx context.Context, caID string) ([]string, error) {
	query := "SELECT id::text FROM certificate_authorities WHERE status='active'"
	args := []any{}
	if caID != "" {
		query += " AND id=$1"
		args = append(args, caID)
	}
	query += " ORDER BY created_at"
	rows, err := s.store.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

// CRLDownloadCAIDs places CAs with matching revocation entries first. A PEM
// bundle can contain multiple CRLs, but common desktop viewers only display
// the first one, so an empty CA must not hide the relevant revocation list.
func (s *Service) CRLDownloadCAIDs(ctx context.Context, filter CRLFilter) ([]string, error) {
	if filter.CAID != "" {
		return s.ActiveCRLCAIDs(ctx, filter.CAID)
	}
	where, args := crlWhere(filter)
	rows, err := s.store.Pool.Query(ctx, `
		SELECT r.ca_id::text
		FROM certificate_revocation_entries r
		JOIN certificate_authorities ca ON ca.id=r.ca_id
		WHERE true`+where+`
		GROUP BY r.ca_id
		ORDER BY max(r.revoked_at) DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matched := []string{}
	seen := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		matched = append(matched, id)
		seen[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if filter.Reason != "" || strings.TrimSpace(filter.Keyword) != "" {
		return matched, nil
	}
	all, err := s.ActiveCRLCAIDs(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, id := range all {
		if !seen[id] {
			matched = append(matched, id)
		}
	}
	return matched, nil
}

func (s *Service) crlSignatureAlgorithm(ctx context.Context, caID string) string {
	if caID == "" {
		return "多 CA 签名算法"
	}
	var cipher string
	if err := s.store.Pool.QueryRow(ctx, "SELECT private_key_ciphertext FROM certificate_authorities WHERE id=$1", caID).Scan(&cipher); err != nil {
		return "-"
	}
	plain, err := s.box.Open(cipher)
	if err != nil {
		return "-"
	}
	block, _ := pem.Decode([]byte(plain))
	if block == nil {
		return "-"
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "-"
	}
	switch signer := key.(type) {
	case *rsa.PrivateKey:
		return "SHA256withRSA"
	case *ecdsa.PrivateKey:
		if signer.Curve.Params().BitSize > 256 {
			return "SHA384withECDSA"
		}
		return "SHA256withECDSA"
	case ed25519.PrivateKey:
		return "Ed25519"
	default:
		return "-"
	}
}
