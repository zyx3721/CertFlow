package router

import (
	"archive/zip"
	"bytes"
	"encoding/pem"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"certflow/backend/internal/repository"
	pkisvc "certflow/backend/internal/service/pki"
)

func (r *Router) previewCertificateCSR(w http.ResponseWriter, q *http.Request) {
	var body pkisvc.RequestInput
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	preview, err := r.pki.GenerateCSR(q.Context(), body)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, preview)
}

func (r *Router) inspectCertificateCSR(w http.ResponseWriter, q *http.Request) {
	var body struct {
		CSRPEM string `json:"csrPEM"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	inspection, err := r.pki.InspectCSR(q.Context(), body.CSRPEM)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, inspection)
}

func (r *Router) caCertificate(w http.ResponseWriter, q *http.Request) {
	item, err := r.pki.CACertificate(q.Context(), q.PathValue("id"))
	if err != nil {
		write(w, http.StatusNotFound, map[string]string{"message": "CA 不存在"})
		return
	}
	write(w, http.StatusOK, map[string]string{
		"filename": sanitizeFilename(item.Name) + ".crt",
		"content":  item.CertificatePEM,
	})
}

func (r *Router) downloadCACertificate(w http.ResponseWriter, q *http.Request) {
	item, err := r.pki.CACertificate(q.Context(), q.PathValue("id"))
	if err != nil {
		write(w, http.StatusNotFound, map[string]string{"message": "CA 不存在"})
		return
	}
	writeCertificateFile(w, sanitizeFilename(item.Name)+".crt", item.CertificatePEM)
}

func (r *Router) deleteCA(w http.ResponseWriter, q *http.Request) {
	if err := r.pki.DeleteCA(q.Context(), q.PathValue("id"), current(q), clientIP(q)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) caDeletionCheck(w http.ResponseWriter, q *http.Request) {
	deletable, err := r.pki.CanDeleteCA(q.Context(), q.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": "CA 不存在"})
		return
	}
	write(w, http.StatusOK, map[string]bool{"deletable": deletable})
}

func (r *Router) certificateTrend(w http.ResponseWriter, q *http.Request) {
	items, err := r.pki.CertificateTrend(q.Context(), current(q))
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取证书趋势失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (r *Router) verifyCertificate(w http.ResponseWriter, q *http.Request) {
	result, err := r.pki.VerifyCertificate(q.Context(), q.PathValue("id"), current(q), clientIP(q))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, result)
}

func (r *Router) deleteCertificate(w http.ResponseWriter, q *http.Request) {
	if err := r.pki.DeleteCertificate(q.Context(), q.PathValue("id"), current(q), clientIP(q)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) downloadCertificate(w http.ResponseWriter, q *http.Request) {
	item, err := r.pki.CertificatePackage(q.Context(), q.PathValue("id"), current(q), clientIP(q))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	base := sanitizeFilename(item.CommonName)
	if item.Status == "rejected" {
		if strings.TrimSpace(item.CSRPEM) == "" {
			write(w, http.StatusBadRequest, map[string]string{"message": "暂无 CSR 信息可下载"})
			return
		}
		if strings.TrimSpace(item.PrivateKeyPEM) == "" {
			writeCSRFile(w, base+".csr", item.CSRPEM)
			return
		}
		archive, err := csrArchive(base, item.CSRPEM, item.PrivateKeyPEM)
		if err != nil {
			write(w, http.StatusInternalServerError, map[string]string{"message": "生成 CSR 压缩包失败"})
			return
		}
		writeArchive(w, base+".zip", archive)
		return
	}
	if item.Status != "valid" || strings.TrimSpace(item.CertificatePEM) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "证书尚未签发，无法下载"})
		return
	}

	if strings.TrimSpace(item.PrivateKeyPEM) == "" {
		writeCertificateFile(w, base+".crt", item.CertificatePEM)
		return
	}

	ca, err := r.pki.CACertificate(q.Context(), item.CAID)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取签发 CA 失败"})
		return
	}
	archive, err := certificateArchive(base, item.CertificatePEM, ca.CertificatePEM, item.PrivateKeyPEM)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "生成证书压缩包失败"})
		return
	}
	writeArchive(w, base+".zip", archive)
}

func writeCertificateFile(w http.ResponseWriter, filename, content string) {
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	_, _ = w.Write([]byte(content))
}

func writeCSRFile(w http.ResponseWriter, filename, content string) {
	w.Header().Set("Content-Type", "application/pkcs10")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	_, _ = w.Write([]byte(content))
}

func writeArchive(w http.ResponseWriter, filename string, archive []byte) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
	_, _ = w.Write(archive)
}

func certificateArchive(base, certificatePEM, issuerPEM, privateKeyPEM string) ([]byte, error) {
	return createArchive([]archiveFile{
		{name: base + "_bundle.crt", content: certificatePEM + "\n" + issuerPEM},
		{name: base + ".key", content: privateKeyPEM},
	})
}

func csrArchive(base, csrPEM, privateKeyPEM string) ([]byte, error) {
	return createArchive([]archiveFile{
		{name: base + ".csr", content: csrPEM},
		{name: base + ".key", content: privateKeyPEM},
	})
}

type archiveFile struct {
	name    string
	content string
}

func createArchive(files []archiveFile) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		entry, err := writer.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write([]byte(file.content)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (r *Router) listCRL(w http.ResponseWriter, q *http.Request) {
	filter, err := parseCRLFilter(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	items, total, err := r.pki.ListCRLEntries(q.Context(), filter)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 CRL 失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{
		"items":    items,
		"total":    total,
		"page":     filter.Page,
		"pageSize": filter.PageSize,
	})
}

func (r *Router) crlMetadata(w http.ResponseWriter, q *http.Request) {
	filter, err := parseCRLFilter(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 CRL 配置失败"})
		return
	}
	interval := int(numberSetting(settings, "crlIntervalHours", 24))
	metadata, err := r.pki.CRLMetadata(q.Context(), interval, filter)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 CRL 元数据失败"})
		return
	}
	metadata["enabled"] = boolSetting(settings, "crlEnabled", true)
	metadata["crlUrl"] = stringSetting(settings, "crlUrl", "/crl")
	write(w, http.StatusOK, metadata)
}

func (r *Router) downloadCRL(w http.ResponseWriter, q *http.Request) {
	filter, err := parseCRLFilter(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 CRL 配置失败"})
		return
	}
	if !boolSetting(settings, "crlEnabled", true) {
		write(w, http.StatusBadRequest, map[string]string{"message": "CRL 服务未启用"})
		return
	}
	caIDs, err := r.pki.CRLDownloadCAIDs(q.Context(), filter)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取签发 CA 失败"})
		return
	}
	if len(caIDs) == 0 {
		write(w, http.StatusNotFound, map[string]string{"message": "没有可下载 CRL 的 CA"})
		return
	}
	interval := int(numberSetting(settings, "crlIntervalHours", 24))
	derFiles := make([][]byte, 0, len(caIDs))
	for _, caID := range caIDs {
		der, err := r.pki.CRLWithFilter(q.Context(), caID, interval, filter)
		if err != nil {
			write(w, http.StatusNotFound, map[string]string{"message": "CRL 不存在"})
			return
		}
		derFiles = append(derFiles, der)
	}
	if err := writeCRLDownload(w, caIDs, derFiles); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "生成 CRL 文件失败"})
	}
}

func parseCRLFilter(q *http.Request) (pkisvc.CRLFilter, error) {
	filter := pkisvc.CRLFilter{
		CAID:     strings.TrimSpace(q.URL.Query().Get("caId")),
		Reason:   strings.TrimSpace(q.URL.Query().Get("reason")),
		Keyword:  strings.TrimSpace(q.URL.Query().Get("keyword")),
		Page:     1,
		PageSize: 10,
	}
	if filter.Reason != "" {
		allowed := map[string]bool{"keyCompromise": true, "cACompromise": true, "affiliationChanged": true, "superseded": true, "cessationOfOperation": true, "unspecified": true}
		if !allowed[filter.Reason] {
			return pkisvc.CRLFilter{}, errors.New("撤销原因不合法")
		}
	}
	if value := q.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return pkisvc.CRLFilter{}, errors.New("页码不合法")
		}
		filter.Page = parsed
	}
	if value := q.URL.Query().Get("pageSize"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return pkisvc.CRLFilter{}, errors.New("每页条数不合法")
		}
		filter.PageSize = parsed
	}
	return filter, nil
}

func writeCRLDownload(w http.ResponseWriter, caIDs []string, derFiles [][]byte) error {
	if len(caIDs) != len(derFiles) || len(caIDs) == 0 {
		return errors.New("invalid CRL download files")
	}
	date := time.Now().Format("2006-01-02")
	if len(caIDs) == 1 {
		content, err := crlPEM(derFiles[0])
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/pkix-crl")
		w.Header().Set("Content-Disposition", `attachment; filename="pki-crl-ca-`+sanitizeFilename(caIDs[0])+`-`+date+`.crl"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		_, _ = w.Write(content)
		return nil
	}
	archive, err := crlDownloadArchive(caIDs, derFiles, date)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="pki-crl-all-`+date+`.zip"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
	_, _ = w.Write(archive)
	return nil
}

func crlDownloadArchive(caIDs []string, derFiles [][]byte, date string) ([]byte, error) {
	if len(caIDs) != len(derFiles) || len(caIDs) < 2 {
		return nil, errors.New("invalid CRL archive files")
	}
	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	for index, der := range derFiles {
		entry, err := writer.Create("pki-crl-ca-" + sanitizeFilename(caIDs[index]) + "-" + date + ".crl")
		if err != nil {
			return nil, err
		}
		if err := pem.Encode(entry, &pem.Block{Type: "X509 CRL", Bytes: der}); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return content.Bytes(), nil
}

func crlPEM(der []byte) ([]byte, error) {
	var content bytes.Buffer
	if err := pem.Encode(&content, &pem.Block{Type: "X509 CRL", Bytes: der}); err != nil {
		return nil, err
	}
	return content.Bytes(), nil
}

func (r *Router) deleteWorkflow(w http.ResponseWriter, q *http.Request) {
	if err := r.pki.DeleteWorkflow(q.Context(), q.PathValue("id"), current(q), clientIP(q)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			status = http.StatusNotFound
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) listAudits(w http.ResponseWriter, q *http.Request) {
	limit, _ := strconv.Atoi(q.URL.Query().Get("limit"))
	items, err := r.store.ListAudits(q.Context(), limit)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取审计日志失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"\"", "-",
		"\r", "-",
		"\n", "-",
	).Replace(value)
	if value == "" {
		return "certificate"
	}
	return value
}

func boolSetting(settings map[string]any, key string, fallback bool) bool {
	value, ok := settings[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func stringSetting(settings map[string]any, key string, fallback string) string {
	value, ok := settings[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func numberSetting(settings map[string]any, key string, fallback float64) float64 {
	value, ok := settings[key].(float64)
	if !ok || value < 1 {
		return fallback
	}
	return value
}
