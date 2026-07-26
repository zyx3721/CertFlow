package pki

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net"
	"strings"
	"time"
	"unicode/utf16"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
	"certflow/backend/internal/security"

	"github.com/google/uuid"
)

type Service struct {
	store    *repository.Store
	box      *security.Cryptobox
	notifier ApprovalNotifier
}

type ApprovalNotifier interface {
	NotifyApproval(context.Context, string, string, time.Time)
}

type RenewalNotifier interface {
	NotifyRenewalResult(context.Context, string, string, *time.Time) error
}

func New(store *repository.Store, box *security.Cryptobox, notifiers ...ApprovalNotifier) *Service {
	var notifier ApprovalNotifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}
	return &Service{store: store, box: box, notifier: notifier}
}

type CreateCAInput struct {
	Name, Type, Algorithm, Subject, ParentID string
	NotAfter                                 time.Time
}
type RequestInput struct {
	CAID, CommonName, Subject, Algorithm, Purpose, CSRPEM, PrivateKeyPEM string
	SAN                                                                  []string
	ValidityDays                                                         int
}

type CSRPreview struct {
	CSRPEM        string `json:"csrPEM"`
	PrivateKeyPEM string `json:"privateKeyPEM"`
}

func revokeReasonLabel(reason string) string {
	return map[string]string{
		"keyCompromise":        "密钥泄露",
		"cACompromise":         "CA 泄露",
		"affiliationChanged":   "归属变更",
		"superseded":           "已替换",
		"cessationOfOperation": "停止运营",
		"unspecified":          "未指定",
	}[reason]
}

func certificateStatusLabel(status string) string {
	return map[string]string{
		"pending":  "待审批",
		"valid":    "有效",
		"revoked":  "已撤销",
		"rejected": "已驳回",
		"expired":  "已过期",
	}[status]
}

func parseSubject(text string) (pkix.Name, error) {
	var n pkix.Name
	for _, part := range strings.Split(text, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(pair[0])) {
		case "CN":
			n.CommonName = pair[1]
		case "O":
			n.Organization = []string{pair[1]}
		case "OU":
			n.OrganizationalUnit = []string{pair[1]}
		case "C":
			n.Country = []string{pair[1]}
		case "ST":
			n.Province = []string{pair[1]}
		case "L":
			n.Locality = []string{pair[1]}
		}
	}
	if n.CommonName == "" {
		return n, errors.New("Subject 必须包含 CN")
	}
	return n, nil
}
func keyFor(algorithm string) (crypto.Signer, error) {
	switch algorithm {
	case "RSA-2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "RSA-4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	case "ECDSA-P256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ECDSA-P384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "ED25519":
		_, key, err := ed25519.GenerateKey(rand.Reader)
		return key, err
	default:
		return nil, errors.New("不支持的密钥算法")
	}
}
func pemKey(key crypto.Signer) (string, error) {
	bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})), nil
}
func pemCert(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
func fingerprintDigest(der []byte) [sha256.Size]byte {
	return sha256.Sum256(der)
}

func fingerprint(der []byte) string {
	sum := fingerprintDigest(der)
	parts := make([]string, len(sum))
	for index, value := range sum {
		parts[index] = strings.ToUpper(hex.EncodeToString([]byte{value}))
	}
	return strings.Join(parts, ":")
}

func certificateRequestFingerprint(commonName, serialNumber, algorithm string) string {
	codeUnits := utf16.Encode([]rune(commonName + serialNumber + algorithm))
	if len(codeUnits) == 0 {
		return ""
	}

	parts := make([]string, 16)
	for index := range parts {
		value := (int(codeUnits[index%len(codeUnits)]) + index*17) % 256
		parts[index] = strings.ToUpper(hex.EncodeToString([]byte{byte(value)}))
	}
	return strings.Join(parts, ":")
}

func subjectKeyID(seed []byte) []byte {
	sum := fingerprintDigest(seed)
	return sum[:20]
}

func (s *Service) CreateCA(ctx context.Context, in CreateCAInput, actor domain.User, ip string) (domain.CA, error) {
	if in.Type != "root" && in.Type != "intermediate" && in.Type != "issuing" {
		return domain.CA{}, errors.New("CA 类型不合法")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.NotAfter.IsZero() {
		return domain.CA{}, errors.New("CA 名称和到期时间不能为空")
	}
	var nameExists bool
	if err := s.store.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM certificate_authorities WHERE name=$1)", in.Name).Scan(&nameExists); err != nil {
		return domain.CA{}, err
	}
	if nameExists {
		return domain.CA{}, errors.New("CA 名称已存在")
	}
	subject, err := parseSubject(in.Subject)
	if err != nil {
		return domain.CA{}, err
	}
	var parentCert *x509.Certificate
	var parentKey crypto.Signer
	if in.Type != "root" {
		if in.ParentID == "" {
			return domain.CA{}, errors.New("非根 CA 必须选择上级")
		}
		var certPEM, ciphertext string
		var parentType string
		err = s.store.Pool.QueryRow(ctx, "SELECT certificate_pem,private_key_ciphertext,type FROM certificate_authorities WHERE id=$1 AND status='active'", in.ParentID).Scan(&certPEM, &ciphertext, &parentType)
		if err != nil {
			return domain.CA{}, errors.New("上级 CA 不存在或未启用")
		}
		if parentType == "issuing" {
			return domain.CA{}, errors.New("签发 CA 不能作为上级")
		}
		block, _ := pem.Decode([]byte(certPEM))
		parentCert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return domain.CA{}, err
		}
		plain, err := s.box.Open(ciphertext)
		if err != nil {
			return domain.CA{}, err
		}
		block, _ = pem.Decode([]byte(plain))
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return domain.CA{}, err
		}
		parentKey = key.(crypto.Signer)
		if in.NotAfter.After(parentCert.NotAfter) {
			return domain.CA{}, errors.New("子 CA 有效期不能超过上级")
		}
	}
	key, err := keyFor(in.Algorithm)
	if err != nil {
		return domain.CA{}, err
	}
	serialText, serial, err := randomSerial()
	if err != nil {
		return domain.CA{}, err
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: subject, NotBefore: time.Now().Add(-5 * time.Minute), NotAfter: in.NotAfter, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature, SubjectKeyId: subjectKeyID([]byte(in.Name + serialText))}
	issuer := template
	signer := key
	if parentCert != nil {
		issuer = parentCert
		signer = parentKey
		template.AuthorityKeyId = parentCert.SubjectKeyId
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, key.Public(), signer)
	if err != nil {
		return domain.CA{}, err
	}
	keyPEM, err := pemKey(key)
	if err != nil {
		return domain.CA{}, err
	}
	ciphertext, err := s.box.Seal(keyPEM)
	if err != nil {
		return domain.CA{}, err
	}
	var id string
	err = s.store.Pool.QueryRow(ctx, "INSERT INTO certificate_authorities(id,name,type,algorithm,subject,parent_id,certificate_pem,private_key_ciphertext,not_before,not_after) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10) RETURNING id::text", uuid.NewString(), in.Name, in.Type, in.Algorithm, in.Subject, in.ParentID, pemCert(der), ciphertext, template.NotBefore, template.NotAfter).Scan(&id)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return domain.CA{}, errors.New("CA 名称已存在")
		}
		return domain.CA{}, err
	}
	if in.Type == "issuing" {
		if _, _, _, err := s.ensureOCSPResponder(ctx, id); err != nil {
			_, _ = s.store.Pool.Exec(ctx, "DELETE FROM certificate_authorities WHERE id=$1", id)
			return domain.CA{}, err
		}
	}
	s.store.Audit(ctx, actor, "创建 CA", in.Name, "pki", "success", ip, "CA "+in.Name+" 已创建")
	return domain.CA{ID: id, Name: in.Name, Type: in.Type, Algorithm: in.Algorithm, Subject: in.Subject, ParentID: in.ParentID, CertificatePEM: pemCert(der), NotBefore: template.NotBefore, NotAfter: template.NotAfter, Status: "active"}, nil
}

func (s *Service) Approve(ctx context.Context, workflowID, status, reason string, actor domain.User, ip string) error {
	var certID, current string
	if err := s.store.Pool.QueryRow(ctx, "SELECT certificate_id::text,status FROM workflows WHERE id=$1", workflowID).Scan(&certID, &current); err != nil || current != "pending" {
		return errors.New("审批记录不存在或已处理")
	}
	if status == "rejected" {
		if strings.TrimSpace(reason) == "" {
			return errors.New("请填写驳回原因")
		}
		_, err := s.store.Pool.Exec(ctx, "UPDATE workflows SET status='rejected',reject_reason=$2,reviewed_by=NULLIF($3,'')::uuid,updated_at=now() WHERE id=$1", workflowID, reason, actor.ID)
		if err != nil {
			return err
		}
		_, err = s.store.Pool.Exec(ctx, "UPDATE certificates SET status='rejected',reject_reason=$2,updated_at=now() WHERE id=$1", certID, reason)
		if err != nil {
			return err
		}
		var commonName string
		if err = s.store.Pool.QueryRow(ctx, "SELECT common_name FROM certificates WHERE id=$1", certID).Scan(&commonName); err != nil {
			return err
		}
		s.store.Audit(ctx, actor, "审批驳回", commonName, "workflows", "success", ip, "审批单 "+workflowID+" 已驳回，原因："+reason)
		return nil
	}
	if status != "approved" {
		return errors.New("审批状态不合法")
	}
	var caID, commonName, subject, algorithm, purpose, csrPEM, certificateKeyCiphertext, ciphertext, caPEM string
	var sanRaw []byte
	var validityEnd time.Time
	var validityDays int
	err := s.store.Pool.QueryRow(ctx, `
		SELECT c.ca_id::text,c.common_name,c.subject,c.algorithm,c.purpose,c.csr_pem,c.private_key_ciphertext,
		       c.san,ca.private_key_ciphertext,ca.certificate_pem,ca.not_after,c.validity_days
		FROM certificates c
		JOIN certificate_authorities ca ON ca.id=c.ca_id
		WHERE c.id=$1 AND c.status='pending'
	`, certID).Scan(&caID, &commonName, &subject, &algorithm, &purpose, &csrPEM, &certificateKeyCiphertext, &sanRaw, &ciphertext, &caPEM, &validityEnd, &validityDays)
	if err != nil {
		return errors.New("待签发证书不存在")
	}
	var san []string
	if err := json.Unmarshal(sanRaw, &san); err != nil {
		return err
	}
	issuerBlock, _ := pem.Decode([]byte(caPEM))
	issuer, err := x509.ParseCertificate(issuerBlock.Bytes)
	if err != nil {
		return err
	}
	plain, err := s.box.Open(ciphertext)
	if err != nil {
		return err
	}
	keyBlock, _ := pem.Decode([]byte(plain))
	keyAny, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}
	issuerKey := keyAny.(crypto.Signer)
	var public crypto.PublicKey
	privateKeyCiphertext := certificateKeyCiphertext
	if csrPEM != "" {
		block, _ := pem.Decode([]byte(csrPEM))
		csr, err := x509.ParseCertificateRequest(block.Bytes)
		if err != nil || csr.CheckSignature() != nil {
			return errors.New("CSR 签名无效")
		}
		public = csr.PublicKey
	} else {
		key, err := keyFor(algorithm)
		if err != nil {
			return err
		}
		public = key.Public()
		privatePEM, err := pemKey(key)
		if err != nil {
			return err
		}
		privateKeyCiphertext, err = s.box.Seal(privatePEM)
		if err != nil {
			return err
		}
	}
	serialText, serial, err := randomSerial()
	if err != nil {
		return err
	}
	name, err := parseSubject(subject)
	if err != nil {
		return err
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: name, NotBefore: time.Now().Add(-5 * time.Minute), NotAfter: time.Now().AddDate(0, 0, validityDays), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, BasicConstraintsValid: true, DNSNames: nil}
	if template.NotAfter.After(validityEnd) {
		template.NotAfter = validityEnd
	}
	for _, value := range san {
		if ip := net.ParseIP(value); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, value)
		}
	}
	if purpose == "server" {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else if purpose == "client" {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return err
	}
	applyCertificateDistributionEndpoints(template, settings, caID)
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, public, issuerKey)
	if err != nil {
		return err
	}
	tx, err := s.store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "UPDATE certificates SET status='valid',serial_number=$2,certificate_pem=$3,private_key_ciphertext=$4,not_before=$5,not_after=$6,updated_at=now() WHERE id=$1", certID, serialText, pemCert(der), privateKeyCiphertext, template.NotBefore, template.NotAfter)
	if err == nil {
		_, err = tx.Exec(ctx, "UPDATE workflows SET status='approved',reviewed_by=NULLIF($2,'')::uuid,updated_at=now() WHERE id=$1", workflowID, actor.ID)
	}
	if err == nil {
		_, err = tx.Exec(ctx, "UPDATE certificate_authorities SET issued_certs=issued_certs+1 WHERE id=$1", caID)
	}
	if err != nil {
		return err
	}
	s.store.Audit(ctx, actor, "审批通过", commonName, "workflows", "success", ip, "审批单 "+workflowID+" 状态更新为 已通过")
	return tx.Commit(ctx)
}
func (s *Service) ListCAs(ctx context.Context) ([]domain.CA, error) {
	rows, err := s.store.Pool.Query(ctx, "SELECT id::text,name,type,algorithm,subject,COALESCE(parent_id::text,''),certificate_pem,not_before,not_after,issued_certs,revoked_certs,status FROM certificate_authorities ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CA
	for rows.Next() {
		var ca domain.CA
		if err := rows.Scan(&ca.ID, &ca.Name, &ca.Type, &ca.Algorithm, &ca.Subject, &ca.ParentID, &ca.CertificatePEM, &ca.NotBefore, &ca.NotAfter, &ca.IssuedCerts, &ca.RevokedCerts, &ca.Status); err != nil {
			return nil, err
		}
		result = append(result, ca)
	}
	return result, rows.Err()
}

func (s *Service) CertificatePackage(ctx context.Context, id string, actor domain.User, ip string) (domain.Certificate, error) {
	var item domain.Certificate
	var cipher string
	var san []byte
	err := s.store.Pool.QueryRow(ctx, `SELECT id::text,ca_id::text,serial_number,common_name,subject,algorithm,purpose,status,applicant,san,COALESCE(not_before,now()),COALESCE(not_after,now()),csr_pem,certificate_pem,private_key_ciphertext FROM certificates WHERE id=$1`, id).Scan(&item.ID, &item.CAID, &item.SerialNumber, &item.CommonName, &item.Subject, &item.Algorithm, &item.Purpose, &item.Status, &item.Applicant, &san, &item.NotBefore, &item.NotAfter, &item.CSRPEM, &item.CertificatePEM, &cipher)
	if err != nil {
		return item, repository.ErrNotFound
	}
	_ = json.Unmarshal(san, &item.SAN)
	item.SerialNumber = displayCertificateSerial(item.SerialNumber)
	item.Source = certificateSource(cipher)
	item.Fingerprint = certificateFingerprint(item.CertificatePEM, item.CommonName, item.SerialNumber, item.Algorithm)
	if strings.TrimSpace(cipher) != "" {
		key, err := s.box.Open(cipher)
		if err != nil {
			return item, err
		}
		item.PrivateKeyPEM = key
	}
	s.store.Audit(ctx, actor, "下载证书材料", item.CommonName, "certificates", "success", ip, certificateDownloadAuditDetail(item.SerialNumber))
	return item, nil
}

func certificateDownloadAuditDetail(serialNumber string) string {
	return "证书 " + serialNumber + " 已下载"
}

func (s *Service) Revoke(ctx context.Context, id, reason string, actor domain.User, ip string) error {
	allowed := map[string]bool{"keyCompromise": true, "cACompromise": true, "affiliationChanged": true, "superseded": true, "cessationOfOperation": true, "unspecified": true}
	if !allowed[reason] {
		return errors.New("撤销原因不合法")
	}
	var caID, name, status, serialNumber, applicantID string
	if err := s.store.Pool.QueryRow(ctx, "SELECT ca_id::text,common_name,status,serial_number,COALESCE(applicant_id::text,'') FROM certificates WHERE id=$1", id).Scan(&caID, &name, &status, &serialNumber, &applicantID); err != nil {
		return repository.ErrNotFound
	}
	if status != "valid" {
		return errors.New("只有有效证书可以撤销")
	}
	tx, err := s.store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "UPDATE certificates SET status='revoked',revoked_at=now(),revoke_reason=$2,updated_at=now() WHERE id=$1", id, reason)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO certificate_revocation_entries(id,certificate_id,ca_id,applicant_id,serial_number,common_name,revoked_at,revoke_reason)
			VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,now(),$7)`, uuid.NewString(), id, caID, applicantID, serialNumber, name, reason)
	}
	if err == nil {
		_, err = tx.Exec(ctx, "UPDATE certificate_authorities SET revoked_certs=revoked_certs+1 WHERE id=$1", caID)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err == nil {
		s.store.Audit(ctx, actor, "撤销证书", name, "certificates", "success", ip, "证书 "+displayCertificateSerial(serialNumber)+" 已撤销，原因："+revokeReasonLabel(reason))
	}
	return err
}

func (s *Service) ListWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	rows, err := s.store.Pool.Query(ctx, "SELECT w.id::text,w.certificate_id::text,c.applicant,c.common_name,w.status,w.reject_reason,w.created_at FROM workflows w JOIN certificates c ON c.id=w.certificate_id ORDER BY w.created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Workflow{}
	for rows.Next() {
		var item domain.Workflow
		if err := rows.Scan(&item.ID, &item.CertificateID, &item.Applicant, &item.CertName, &item.Status, &item.RejectReason, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
