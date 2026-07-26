package pki

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"strings"
)

type CSRSubject struct {
	CommonName string `json:"commonName"`
	Org        string `json:"org"`
	OrgUnit    string `json:"orgUnit"`
	Country    string `json:"country"`
	Province   string `json:"province"`
	Locality   string `json:"locality"`
}

type CSRInspection struct {
	Subject   CSRSubject `json:"subject"`
	SAN       []string   `json:"san"`
	Algorithm string     `json:"algorithm"`
}

// GenerateCSR creates a PKCS#10 request for the system-generated request flow.
// The private key is returned only to the requesting browser and is encrypted when
// the application is submitted.
func (s *Service) GenerateCSR(ctx context.Context, in RequestInput) (CSRPreview, error) {
	_ = ctx
	subject, err := parseSubject(in.Subject)
	if err != nil {
		return CSRPreview{}, err
	}
	for _, value := range in.SAN {
		if strings.TrimSpace(value) == "" {
			return CSRPreview{}, errors.New("证书域名不能为空")
		}
	}
	key, err := keyFor(in.Algorithm)
	if err != nil {
		return CSRPreview{}, err
	}
	request := &x509.CertificateRequest{Subject: subject}
	for _, value := range in.SAN {
		value = strings.TrimSpace(value)
		if ip := net.ParseIP(value); ip != nil {
			request.IPAddresses = append(request.IPAddresses, ip)
		} else {
			request.DNSNames = append(request.DNSNames, value)
		}
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, request, key)
	if err != nil {
		return CSRPreview{}, err
	}
	privateKeyPEM, err := pemKey(key)
	if err != nil {
		return CSRPreview{}, err
	}
	return CSRPreview{
		CSRPEM:        string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})),
		PrivateKeyPEM: privateKeyPEM,
	}, nil
}

func (s *Service) InspectCSR(ctx context.Context, csrPEM string) (CSRInspection, error) {
	_ = ctx
	csr, err := parseCertificateRequest(csrPEM)
	if err != nil {
		return CSRInspection{}, err
	}
	algorithm, err := csrAlgorithm(csr.PublicKey)
	if err != nil {
		return CSRInspection{}, err
	}
	subject := csr.Subject
	inspection := CSRInspection{
		Subject: CSRSubject{
			CommonName: subject.CommonName,
			Org:        firstSubjectValue(subject.Organization),
			OrgUnit:    firstSubjectValue(subject.OrganizationalUnit),
			Country:    firstSubjectValue(subject.Country),
			Province:   firstSubjectValue(subject.Province),
			Locality:   firstSubjectValue(subject.Locality),
		},
		Algorithm: algorithm,
	}
	inspection.SAN = append(inspection.SAN, csr.DNSNames...)
	for _, ip := range csr.IPAddresses {
		inspection.SAN = append(inspection.SAN, ip.String())
	}
	return inspection, nil
}

func validateCSRKeyPair(csrPEM, privateKeyPEM string) error {
	csrPEM = strings.TrimSpace(csrPEM)
	privateKeyPEM = strings.TrimSpace(privateKeyPEM)
	if csrPEM == "" && privateKeyPEM == "" {
		return nil
	}
	if csrPEM == "" {
		return errors.New("私钥必须与 CSR 一同提交")
	}
	csr, err := parseCertificateRequest(csrPEM)
	if err != nil {
		return err
	}
	if privateKeyPEM == "" {
		return nil
	}
	keyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	if keyBlock == nil {
		return errors.New("私钥格式不合法")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return errors.New("私钥格式不合法")
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return errors.New("私钥格式不合法")
	}
	csrPublic, err := x509.MarshalPKIXPublicKey(csr.PublicKey)
	if err != nil {
		return err
	}
	keyPublic, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return err
	}
	if !bytes.Equal(csrPublic, keyPublic) {
		return errors.New("私钥与 CSR 不匹配")
	}
	return nil
}

func parseCertificateRequest(csrPEM string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(csrPEM)))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, errors.New("CSR 格式不合法")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || csr.CheckSignature() != nil {
		return nil, errors.New("CSR 签名无效")
	}
	return csr, nil
}

func firstSubjectValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func csrAlgorithm(publicKey crypto.PublicKey) (string, error) {
	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		switch key.N.BitLen() {
		case 2048:
			return "RSA-2048", nil
		case 4096:
			return "RSA-4096", nil
		default:
			return "", errors.New("手动 CSR 仅支持 RSA-2048 或 RSA-4096")
		}
	case *ecdsa.PublicKey:
		switch key.Curve.Params().Name {
		case "P-256":
			return "ECDSA-P256", nil
		case "P-384":
			return "ECDSA-P384", nil
		default:
			return "", errors.New("手动 CSR 仅支持 ECDSA-P256 或 ECDSA-P384")
		}
	case ed25519.PublicKey:
		return "ED25519", nil
	default:
		return "", errors.New("不支持的 CSR 公钥算法")
	}
}
