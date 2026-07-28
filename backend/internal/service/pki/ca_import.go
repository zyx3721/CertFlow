package pki

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"

	"github.com/google/uuid"
)

// ImportCAInput contains material supplied by an administrator. CSRPEM is
// optional because CA CSRs are not retained by the certificate-authority model.
type ImportCAInput struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	ParentID       string `json:"parentId"`
	CertificatePEM string `json:"certificatePEM"`
	PrivateKeyPEM  string `json:"privateKeyPEM"`
	CSRPEM         string `json:"csrPEM"`
}

func (s *Service) ImportCA(ctx context.Context, in ImportCAInput, actor domain.User, ip string) (domain.CA, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.ParentID = strings.TrimSpace(in.ParentID)
	if in.Name == "" {
		return domain.CA{}, errors.New("CA 名称不能为空")
	}
	if in.Type != "root" && in.Type != "intermediate" && in.Type != "issuing" {
		return domain.CA{}, errors.New("CA 类型不合法")
	}
	if in.Type == "root" && in.ParentID != "" {
		return domain.CA{}, errors.New("根 CA 不能设置上级 CA")
	}
	if in.Type != "root" && in.ParentID == "" {
		return domain.CA{}, errors.New("非根 CA 必须选择上级 CA")
	}

	certificate, err := parseImportedCACertificate(in.CertificatePEM)
	if err != nil {
		return domain.CA{}, err
	}
	key, err := parseImportedCAPrivateKey(in.PrivateKeyPEM)
	if err != nil {
		return domain.CA{}, err
	}
	if !samePublicKey(certificate.PublicKey, key.Public()) {
		return domain.CA{}, errors.New("CA 证书与私钥不匹配")
	}
	if err := validateImportedCA(certificate); err != nil {
		return domain.CA{}, err
	}
	if strings.TrimSpace(in.CSRPEM) != "" {
		if err := validateImportedCACSR(in.CSRPEM, certificate.PublicKey); err != nil {
			return domain.CA{}, err
		}
	}

	var nameExists bool
	if err := s.store.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM certificate_authorities WHERE name=$1)", in.Name).Scan(&nameExists); err != nil {
		return domain.CA{}, err
	}
	if nameExists {
		return domain.CA{}, errors.New("CA 名称已存在")
	}

	if in.Type == "root" {
		if !bytes.Equal(certificate.RawSubject, certificate.RawIssuer) || certificate.CheckSignatureFrom(certificate) != nil {
			return domain.CA{}, errors.New("根 CA 必须是有效的自签名证书")
		}
	} else if err := s.validateImportedCAParent(ctx, in.ParentID, certificate); err != nil {
		return domain.CA{}, err
	}

	algorithm, err := importedCAAlgorithm(key)
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
	err = s.store.Pool.QueryRow(ctx, `
		INSERT INTO certificate_authorities(
			id,name,type,algorithm,subject,parent_id,certificate_pem,private_key_ciphertext,not_before,not_after
		) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10)
		RETURNING id::text
	`, uuid.NewString(), in.Name, in.Type, algorithm, certificate.Subject.String(), in.ParentID,
		pemCert(certificate.Raw), ciphertext, certificate.NotBefore, certificate.NotAfter).Scan(&id)
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

	s.store.Audit(ctx, actor, "导入 CA", in.Name, "pki", "success", ip, "CA "+in.Name+" 已导入")
	return domain.CA{
		ID:             id,
		Name:           in.Name,
		Type:           in.Type,
		Algorithm:      algorithm,
		Subject:        certificate.Subject.String(),
		ParentID:       in.ParentID,
		CertificatePEM: pemCert(certificate.Raw),
		NotBefore:      certificate.NotBefore,
		NotAfter:       certificate.NotAfter,
		Status:         "active",
	}, nil
}

func (s *Service) validateImportedCAParent(ctx context.Context, parentID string, certificate *x509.Certificate) error {
	var certificatePEM, parentType string
	err := s.store.Pool.QueryRow(ctx, `
		SELECT certificate_pem,type
		FROM certificate_authorities
		WHERE id=$1 AND status='active'
	`, parentID).Scan(&certificatePEM, &parentType)
	if err != nil {
		return errors.New("上级 CA 不存在或未启用")
	}
	if parentType == "issuing" {
		return errors.New("签发 CA 不能作为上级")
	}
	parent, err := parseImportedCACertificate(certificatePEM)
	if err != nil {
		return errors.New("上级 CA 证书无效")
	}
	if certificate.CheckSignatureFrom(parent) != nil {
		return errors.New("CA 证书未由所选上级 CA 签发")
	}
	if certificate.NotAfter.After(parent.NotAfter) {
		return errors.New("CA 有效期不能超过上级 CA")
	}
	return nil
}

func parseImportedCACertificate(value string) (*x509.Certificate, error) {
	block, rest := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil || block.Type != "CERTIFICATE" || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("CA 证书 PEM 格式不正确")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("CA 证书 PEM 格式不正确")
	}
	return certificate, nil
}

func parseImportedCAPrivateKey(value string) (crypto.Signer, error) {
	block, rest := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("CA 私钥 PEM 格式不正确")
	}
	if x509.IsEncryptedPEMBlock(block) {
		return nil, errors.New("不支持加密的 CA 私钥，请先在受控环境中转换为未加密 PEM")
	}

	var key any
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, errors.New("不支持的 CA 私钥 PEM 类型")
	}
	if err != nil {
		return nil, errors.New("CA 私钥 PEM 格式不正确")
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.New("CA 私钥类型不支持")
	}
	return signer, nil
}

func validateImportedCA(certificate *x509.Certificate) error {
	if !certificate.IsCA || !certificate.BasicConstraintsValid {
		return errors.New("证书不是有效的 CA 证书")
	}
	if certificate.KeyUsage&x509.KeyUsageCertSign == 0 || certificate.KeyUsage&x509.KeyUsageCRLSign == 0 {
		return errors.New("CA 证书必须具备证书签发和 CRL 签名用途")
	}
	now := time.Now()
	if certificate.NotAfter.Before(now) {
		return errors.New("CA 证书已过期")
	}
	if certificate.NotBefore.After(now) {
		return errors.New("CA 证书尚未生效")
	}
	return nil
}

func validateImportedCACSR(value string, expectedPublicKey crypto.PublicKey) error {
	block, rest := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil || block.Type != "CERTIFICATE REQUEST" || len(strings.TrimSpace(string(rest))) != 0 {
		return errors.New("CA CSR PEM 格式不正确")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || csr.CheckSignature() != nil {
		return errors.New("CA CSR 签名无效")
	}
	if !samePublicKey(csr.PublicKey, expectedPublicKey) {
		return errors.New("CA CSR 与证书公钥不匹配")
	}
	return nil
}

func samePublicKey(left, right crypto.PublicKey) bool {
	leftDER, leftErr := x509.MarshalPKIXPublicKey(left)
	rightDER, rightErr := x509.MarshalPKIXPublicKey(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftDER, rightDER)
}

func importedCAAlgorithm(key crypto.Signer) (string, error) {
	switch signer := key.(type) {
	case *rsa.PrivateKey:
		switch signer.N.BitLen() {
		case 2048:
			return "RSA-2048", nil
		case 4096:
			return "RSA-4096", nil
		default:
			return "", errors.New("仅支持 RSA-2048 或 RSA-4096 CA 私钥")
		}
	case *ecdsa.PrivateKey:
		switch signer.Curve.Params().Name {
		case "P-256":
			return "ECDSA-P256", nil
		case "P-384":
			return "ECDSA-P384", nil
		default:
			return "", errors.New("仅支持 ECDSA-P256 或 ECDSA-P384 CA 私钥")
		}
	case ed25519.PrivateKey:
		return "ED25519", nil
	default:
		return "", errors.New("不支持的 CA 私钥算法")
	}
}
