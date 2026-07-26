package pki

import (
	"crypto/x509/pkix"
	"errors"
	"net"
	"strings"
)

func validateRequestInput(in *RequestInput, subject pkix.Name) error {
	if in.ValidityDays < 1 || in.ValidityDays > 7300 {
		return errors.New("证书有效期必须在 1 到 7300 天之间")
	}
	if in.Purpose != "server" && in.Purpose != "client" && in.Purpose != "mtls" {
		return errors.New("证书用途不合法")
	}
	switch in.Algorithm {
	case "RSA-2048", "RSA-4096", "ECDSA-P256", "ECDSA-P384", "ED25519":
	default:
		return errors.New("不支持的密钥算法")
	}
	if len(in.SAN) == 0 {
		return errors.New("证书域名不能为空")
	}
	if len(in.SAN) > 100 {
		return errors.New("SAN 数量不能超过 100")
	}
	in.CommonName = strings.TrimSpace(in.CommonName)
	subject.CommonName = strings.TrimSpace(subject.CommonName)
	if in.CommonName == "" {
		in.CommonName = subject.CommonName
	}
	if in.CommonName == "" {
		return errors.New("通用名称不能为空")
	}
	if in.CommonName != subject.CommonName {
		return errors.New("通用名称必须与 Subject 中的 CN 一致")
	}
	if len(subject.Country) > 0 && len(strings.TrimSpace(subject.Country[0])) != 2 {
		return errors.New("国家代码需为 2 位，例如 CN")
	}
	for index, value := range in.SAN {
		value = strings.TrimSpace(value)
		if value == "" {
			return errors.New("证书域名不能为空")
		}
		if strings.Trim(value, "0123456789.") == "" && net.ParseIP(value) == nil {
			return errors.New("IP 地址格式不合法")
		}
		in.SAN[index] = value
	}
	return nil
}
