package router

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"certflow/backend/config"
	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
	authsvc "certflow/backend/internal/service/auth"
	notifysvc "certflow/backend/internal/service/notify"
	pkisvc "certflow/backend/internal/service/pki"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/crypto/ocsp"
)

type Router struct {
	cfg    config.Config
	store  *repository.Store
	auth   *authsvc.Service
	pki    *pkisvc.Service
	notify *notifysvc.Service
	logger *slog.Logger
}

func New(cfg config.Config, store *repository.Store, auth *authsvc.Service, pki *pkisvc.Service, notify *notifysvc.Service, logger *slog.Logger) http.Handler {
	r := &Router{cfg: cfg, store: store, auth: auth, pki: pki, notify: notify, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", r.health)
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)
	mux.HandleFunc("GET /crl/", r.crl)
	mux.HandleFunc("GET /ocsp", r.ocspStatus)
	mux.HandleFunc("POST /ocsp", r.ocsp)
	mux.HandleFunc("GET /ocsp/health", r.ocspHealth)
	mux.HandleFunc("GET /ocsp/{serial}", r.ocspStatusBySerial)
	mux.HandleFunc("POST /api/v1/auth/login", r.login)
	mux.HandleFunc("GET /api/v1/auth/providers", r.authProviders)
	mux.HandleFunc("GET /api/v1/public/settings", r.publicSettings)
	mux.HandleFunc("GET /api/v1/auth/password-reset/captcha", r.passwordResetCaptcha)
	mux.HandleFunc("POST /api/v1/auth/password-reset/verify", r.passwordResetVerify)
	mux.HandleFunc("POST /api/v1/auth/password-reset/send", r.passwordResetSend)
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", r.passwordResetConfirm)
	mux.HandleFunc("POST /api/v1/auth/logout", r.require(r.logout))
	mux.HandleFunc("GET /api/v1/auth/me", r.require(r.me))
	mux.HandleFunc("GET /api/v1/cas", r.requirePermission(domain.PermissionCARead, r.listCAs))
	mux.HandleFunc("POST /api/v1/cas", r.requirePermission(domain.PermissionCAAdd, r.createCA))
	mux.HandleFunc("GET /api/v1/cas/{id}/certificate", r.requirePermission(domain.PermissionCARead, r.caCertificate))
	mux.HandleFunc("GET /api/v1/cas/{id}/download", r.requirePermission(domain.PermissionCADownload, r.downloadCACertificate))
	mux.HandleFunc("GET /api/v1/cas/{id}/deletion-check", r.requirePermission(domain.PermissionCADelete, r.caDeletionCheck))
	mux.HandleFunc("DELETE /api/v1/cas/{id}", r.requirePermission(domain.PermissionCADelete, r.deleteCA))
	mux.HandleFunc("GET /api/v1/certificates", r.requirePermission(domain.PermissionCertificateRead, r.listCertificates))
	mux.HandleFunc("POST /api/v1/certificates/csr-preview", r.requirePermission(domain.PermissionCertificateRequest, r.previewCertificateCSR))
	mux.HandleFunc("POST /api/v1/certificates/csr-inspect", r.requirePermission(domain.PermissionCertificateRequest, r.inspectCertificateCSR))
	mux.HandleFunc("POST /api/v1/certificates", r.requirePermission(domain.PermissionCertificateRequest, r.requestCertificate))
	mux.HandleFunc("GET /api/v1/certificates/trend", r.requirePermission(domain.PermissionDashboardRead, r.certificateTrend))
	mux.HandleFunc("GET /api/v1/certificates/{id}/package", r.requirePermission(domain.PermissionCertificateDownload, r.certificatePackage))
	mux.HandleFunc("GET /api/v1/certificates/{id}/download", r.requirePermission(domain.PermissionCertificateDownload, r.downloadCertificate))
	mux.HandleFunc("GET /api/v1/certificates/{id}/verify", r.requirePermission(domain.PermissionCertificateVerify, r.verifyCertificate))
	mux.HandleFunc("POST /api/v1/certificates/{id}/revoke", r.requirePermission(domain.PermissionCertificateRevoke, r.revokeCertificate))
	mux.HandleFunc("DELETE /api/v1/certificates/{id}", r.requirePermission(domain.PermissionCertificateDelete, r.deleteCertificate))
	mux.HandleFunc("GET /api/v1/crl", r.requirePermission(domain.PermissionCRLRead, r.listCRL))
	mux.HandleFunc("GET /api/v1/crl/metadata", r.requirePermission(domain.PermissionCRLRead, r.crlMetadata))
	mux.HandleFunc("GET /api/v1/crl/download", r.requirePermission(domain.PermissionCRLManage, r.downloadCRL))
	mux.HandleFunc("GET /api/v1/workflows", r.requirePermission(domain.PermissionWorkflowRead, r.listWorkflows))
	mux.HandleFunc("PATCH /api/v1/workflows/{id}", r.requirePermission(domain.PermissionWorkflowApprove, r.updateWorkflow))
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", r.requirePermission(domain.PermissionWorkflowDelete, r.deleteWorkflow))
	mux.HandleFunc("GET /api/v1/audit-logs", r.requirePermission(domain.PermissionAuditRead, r.listAudits))
	mux.HandleFunc("POST /api/v1/users/change-password", r.require(r.changePassword))
	mux.HandleFunc("GET /api/v1/settings", r.requirePermission(domain.PermissionSettingsBaseRead, r.getSettings))
	mux.HandleFunc("PUT /api/v1/settings", r.requirePermission(domain.PermissionSettingsBaseManage, r.saveSettings))
	mux.HandleFunc("GET /api/v1/settings/permissions", r.requirePermission(domain.PermissionSettingsUsersRead, r.listPermissions))
	mux.HandleFunc("GET /api/v1/settings/roles", r.requirePermission(domain.PermissionSettingsUsersRead, r.listRoles))
	mux.HandleFunc("POST /api/v1/settings/roles", r.requirePermission(domain.PermissionSettingsUsersManage, r.createRole))
	mux.HandleFunc("PUT /api/v1/settings/roles/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.updateRole))
	mux.HandleFunc("POST /api/v1/settings/roles/{id}/disabled", r.requirePermission(domain.PermissionSettingsUsersManage, r.setRoleDisabled))
	mux.HandleFunc("DELETE /api/v1/settings/roles/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.deleteRole))
	mux.HandleFunc("GET /api/v1/settings/user-groups", r.requirePermission(domain.PermissionSettingsUsersRead, r.listUserGroups))
	mux.HandleFunc("POST /api/v1/settings/user-groups", r.requirePermission(domain.PermissionSettingsUsersManage, r.createUserGroup))
	mux.HandleFunc("PUT /api/v1/settings/user-groups/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.updateUserGroup))
	mux.HandleFunc("DELETE /api/v1/settings/user-groups/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.deleteUserGroup))
	mux.HandleFunc("GET /api/v1/settings/users", r.requirePermission(domain.PermissionSettingsUsersRead, r.listUsers))
	mux.HandleFunc("POST /api/v1/settings/users", r.requirePermission(domain.PermissionSettingsUsersManage, r.createManagedUser))
	mux.HandleFunc("PUT /api/v1/settings/users/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.updateManagedUser))
	mux.HandleFunc("POST /api/v1/settings/users/{id}/disabled", r.requirePermission(domain.PermissionSettingsUsersManage, r.setSettingsUserStatus))
	mux.HandleFunc("DELETE /api/v1/settings/users/{id}", r.requirePermission(domain.PermissionSettingsUsersManage, r.deleteUser))
	mux.HandleFunc("GET /api/v1/settings/auth-provider", r.requirePermission(domain.PermissionSettingsAuthRead, r.getAuthProvider))
	mux.HandleFunc("PUT /api/v1/settings/auth-provider", r.requirePermission(domain.PermissionSettingsAuthManage, r.saveAuthProvider))
	mux.HandleFunc("POST /api/v1/settings/auth-provider/test", r.requirePermission(domain.PermissionSettingsAuthManage, r.testAuthProvider))
	mux.HandleFunc("GET /api/v1/settings/email", r.requirePermission(domain.PermissionSettingsNotifyRead, r.getEmailSetting))
	mux.HandleFunc("PUT /api/v1/settings/email", r.requirePermission(domain.PermissionSettingsNotifyManage, r.saveEmailSetting))
	mux.HandleFunc("POST /api/v1/settings/email/test", r.requirePermission(domain.PermissionSettingsNotifyManage, r.testEmailSetting))
	mux.HandleFunc("GET /api/v1/settings/notifications", r.requirePermission(domain.PermissionSettingsNotifyRead, r.listNotificationChannels))
	mux.HandleFunc("PUT /api/v1/settings/notifications/{id}", r.requirePermission(domain.PermissionSettingsNotifyManage, r.saveNotificationChannel))
	mux.HandleFunc("POST /api/v1/settings/notifications/{id}/test", r.requirePermission(domain.PermissionSettingsNotifyManage, r.testNotificationChannel))
	return r.cors(r.log(mux))
}

type contextKey string

const userKey contextKey = "user"

func (r *Router) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.cfg.CORS.Origin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if q.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, q)
	})
}
func (r *Router) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, q)
		r.logger.Info("HTTP request completed", "method", q.Method, "path", q.URL.Path, "duration", time.Since(started).String())
	})
}
func write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func decode(q *http.Request, target any) error {
	q.Body = http.MaxBytesReader(nil, q.Body, 1<<20)
	de := json.NewDecoder(q.Body)
	de.DisallowUnknownFields()
	return de.Decode(target)
}
func bearer(q *http.Request) string {
	value := strings.TrimSpace(q.Header.Get("Authorization"))
	if parts := strings.Fields(value); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func (r *Router) require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, q *http.Request) {
		user, err := r.auth.Authenticate(q.Context(), bearer(q))
		if err != nil {
			write(w, 401, map[string]string{"message": "请先登录"})
			return
		}
		next.ServeHTTP(w, q.WithContext(context.WithValue(q.Context(), userKey, user)))
	}
}
func (r *Router) requirePermission(permission string, next http.HandlerFunc) http.HandlerFunc {
	return r.require(func(w http.ResponseWriter, q *http.Request) {
		user := q.Context().Value(userKey).(domain.User)
		for _, item := range user.Permissions {
			if item == permission {
				next(w, q)
				return
			}
		}
		write(w, 403, map[string]string{"message": "权限不足"})
	})
}
func current(q *http.Request) domain.User { return q.Context().Value(userKey).(domain.User) }
func clientIP(q *http.Request) string {
	remoteIP := requestIP(q.RemoteAddr)
	if !isLoopbackIP(remoteIP) {
		return remoteIP
	}

	for _, value := range strings.Split(q.Header.Get("X-Forwarded-For"), ",") {
		if forwardedIP := validIP(value); forwardedIP != "" {
			return forwardedIP
		}
	}
	if realIP := validIP(q.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	return remoteIP
}

func requestIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func isLoopbackIP(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && ip.IsLoopback()
}

func validIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}
func (r *Router) health(w http.ResponseWriter, q *http.Request) {
	write(w, 200, map[string]any{"ok": true, "service": "certflow", "time": time.Now().UTC()})
}
func (r *Router) login(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Provider string `json:"provider"`
	}
	if err := decode(q, &body); err != nil {
		write(w, 400, map[string]string{"message": "请求格式不正确"})
		return
	}
	session, err := r.auth.Login(q.Context(), body.Username, body.Password, body.Provider)
	if err != nil {
		write(w, 401, map[string]string{"message": loginFailureMessage(err)})
		return
	}
	r.store.Audit(q.Context(), session.User, "用户登录", session.User.Username, "auth", "success", clientIP(q), authenticationAuditDetail(session.User, "登录"))
	write(w, 200, map[string]any{"token": session.Token, "expiresAt": session.ExpiresAt, "user": session.User})
}

func loginFailureMessage(err error) string {
	if errors.Is(err, authsvc.ErrUserNotProvisioned) {
		return "用户未在平台中启用"
	}
	return "用户名或密码错误"
}

func (r *Router) authProviders(w http.ResponseWriter, q *http.Request) {
	items, resetEnabled, err := r.auth.PublicProviders(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取登录方式失败"})
		return
	}
	write(w, http.StatusOK, authProvidersResponse(items, resetEnabled))
}

func authProvidersResponse(items []map[string]any, resetEnabled bool) map[string]any {
	return map[string]any{
		"items":                items,
		"total":                len(items),
		"passwordResetEnabled": resetEnabled,
	}
}
func (r *Router) logout(w http.ResponseWriter, q *http.Request) {
	user := current(q)
	_ = r.auth.Logout(q.Context(), bearer(q))
	r.store.Audit(q.Context(), user, "用户注销", user.Username, "auth", "success", clientIP(q), authenticationAuditDetail(user, "注销"))
	write(w, 204, nil)
}

func authenticationAuditDetail(user domain.User, action string) string {
	source := "本地用户"
	if user.AuthenticationProvider == "ldap" {
		source = "LDAP 用户"
	}
	return source + " " + user.Username + " " + action + "成功"
}

func (r *Router) me(w http.ResponseWriter, q *http.Request) { write(w, 200, current(q)) }
func (r *Router) listCAs(w http.ResponseWriter, q *http.Request) {
	items, err := r.pki.ListCAs(q.Context())
	if err != nil {
		write(w, 500, map[string]string{"message": "读取 CA 失败"})
		return
	}
	write(w, 200, map[string]any{"items": items, "total": len(items)})
}
func (r *Router) createCA(w http.ResponseWriter, q *http.Request) {
	var body pkisvc.CreateCAInput
	if err := decode(q, &body); err != nil {
		write(w, 400, map[string]string{"message": "请求格式不正确"})
		return
	}
	ca, err := r.pki.CreateCA(q.Context(), body, current(q), clientIP(q))
	if err != nil {
		write(w, 400, map[string]string{"message": err.Error()})
		return
	}
	write(w, 201, ca)
}
func (r *Router) requestCertificate(w http.ResponseWriter, q *http.Request) {
	var body pkisvc.RequestInput
	if err := decode(q, &body); err != nil {
		write(w, 400, map[string]string{"message": "请求格式不正确"})
		return
	}
	cert, err := r.pki.CreateRequest(q.Context(), body, current(q), clientIP(q))
	if err != nil {
		write(w, 400, map[string]string{"message": err.Error()})
		return
	}
	write(w, 201, cert)
}
func (r *Router) listCertificates(w http.ResponseWriter, q *http.Request) {
	items, err := r.pki.ListCertificates(q.Context())
	if err != nil {
		write(w, 500, map[string]string{"message": "读取证书失败"})
		return
	}
	write(w, 200, map[string]any{"items": items, "total": len(items)})
}
func (r *Router) certificatePackage(w http.ResponseWriter, q *http.Request) {
	id := q.PathValue("id")
	if id == "" {
		write(w, 404, map[string]string{"message": "证书不存在"})
		return
	}
	item, err := r.pki.CertificatePackage(q.Context(), id, current(q), clientIP(q))
	if err != nil {
		status := 400
		if errors.Is(err, repository.ErrNotFound) {
			status = 404
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	write(w, 200, item)
}
func (r *Router) revokeCertificate(w http.ResponseWriter, q *http.Request) {
	id := q.PathValue("id")
	if id == "" {
		write(w, 404, map[string]string{"message": "证书不存在"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decode(q, &body); err != nil {
		write(w, 400, map[string]string{"message": "请求格式不正确"})
		return
	}
	if err := r.pki.Revoke(q.Context(), id, body.Reason, current(q), clientIP(q)); err != nil {
		write(w, 400, map[string]string{"message": err.Error()})
		return
	}
	write(w, 200, map[string]string{"status": "ok"})
}
func (r *Router) listWorkflows(w http.ResponseWriter, q *http.Request) {
	items, err := r.pki.ListWorkflows(q.Context())
	if err != nil {
		write(w, 500, map[string]string{"message": "读取审批失败"})
		return
	}
	write(w, 200, map[string]any{"items": items, "total": len(items)})
}
func (r *Router) listUsers(w http.ResponseWriter, q *http.Request) {
	items, err := r.store.ListUsers(q.Context())
	if err != nil {
		write(w, 500, map[string]string{"message": "读取用户失败"})
		return
	}
	write(w, 200, map[string]any{"items": items, "total": len(items)})
}
func (r *Router) crl(w http.ResponseWriter, q *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(q.URL.Path, "/crl/"), ".crl")
	if id == "" {
		write(w, 404, map[string]string{"message": "CRL 不存在"})
		return
	}
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 CRL 配置失败"})
		return
	}
	if !boolSetting(settings, "crlEnabled", true) {
		write(w, http.StatusNotFound, map[string]string{"message": "CRL 服务未启用"})
		return
	}
	der, err := r.pki.CRLWithInterval(q.Context(), id, int(numberSetting(settings, "crlIntervalHours", 24)))
	if err != nil {
		write(w, 404, map[string]string{"message": "CRL 不存在"})
		return
	}
	writeCRLFile(w, id, der)
}

func writeCRLFile(w http.ResponseWriter, id string, der []byte) {
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Header().Set("Content-Disposition", "attachment; filename=ca-"+id+".crl")
	_, _ = w.Write(pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}))
}
func (r *Router) ocsp(w http.ResponseWriter, q *http.Request) {
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		http.Error(w, "OCSP responder unavailable", http.StatusInternalServerError)
		return
	}
	isRFCRequest := strings.Contains(strings.ToLower(q.Header.Get("Content-Type")), "application/ocsp-request")
	if !boolSetting(settings, "ocspEnabled", true) {
		if isRFCRequest {
			w.Header().Set("Content-Type", "application/ocsp-response")
			_, _ = w.Write(ocsp.UnauthorizedErrorResponse)
			return
		}
		r.ocspJSONRequest(w, q, false)
		return
	}
	if !isRFCRequest {
		r.ocspJSONRequest(w, q, true)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, q.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid OCSP request", http.StatusBadRequest)
		return
	}
	response, err := r.pki.OCSP(q.Context(), body)
	if err != nil {
		http.Error(w, "OCSP responder unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/ocsp-response")
	_, _ = w.Write(response)
}

func (r *Router) ocspHealth(w http.ResponseWriter, q *http.Request) {
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 OCSP 配置失败"})
		return
	}
	write(w, http.StatusOK, ocspHealthResponse{OK: true, Enabled: boolSetting(settings, "ocspEnabled", true)})
}

func (r *Router) ocspStatus(w http.ResponseWriter, q *http.Request) {
	serial := q.URL.Query().Get("serialNumber")
	if serial == "" {
		serial = q.URL.Query().Get("serial")
	}
	r.writeOCSPStatus(w, q, serial)
}

func (r *Router) ocspStatusBySerial(w http.ResponseWriter, q *http.Request) {
	r.writeOCSPStatus(w, q, q.PathValue("serial"))
}

func (r *Router) writeOCSPStatus(w http.ResponseWriter, q *http.Request, serial string) {
	if strings.TrimSpace(serial) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "serialNumber 不能为空"})
		return
	}
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取 OCSP 配置失败"})
		return
	}
	if !boolSetting(settings, "ocspEnabled", true) {
		write(w, http.StatusOK, ocspDisabledStatus(serial))
		return
	}
	result, err := r.pki.OCSPStatus(q.Context(), serial)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "查询 OCSP 状态失败"})
		return
	}
	write(w, http.StatusOK, result)
}

func (r *Router) ocspJSONRequest(w http.ResponseWriter, q *http.Request, enabled bool) {
	var body struct {
		SerialNumber   string `json:"serialNumber"`
		Serial         string `json:"serial"`
		CertificatePEM string `json:"certificatePem"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	serial := body.SerialNumber
	if serial == "" {
		serial = body.Serial
	}
	if serial == "" && strings.TrimSpace(body.CertificatePEM) != "" {
		block, _ := pem.Decode([]byte(body.CertificatePEM))
		if block == nil {
			write(w, http.StatusBadRequest, map[string]string{"message": "证书 PEM 格式不正确"})
			return
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			write(w, http.StatusBadRequest, map[string]string{"message": "证书 PEM 格式不正确"})
			return
		}
		serial = strings.ToUpper(certificate.SerialNumber.Text(16))
	}
	if strings.TrimSpace(serial) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "serialNumber 或 certificatePem 不能为空"})
		return
	}
	if !enabled {
		write(w, http.StatusOK, ocspDisabledStatus(serial))
		return
	}
	result, err := r.pki.OCSPStatus(q.Context(), serial)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "查询 OCSP 状态失败"})
		return
	}
	write(w, http.StatusOK, result)
}

func ocspDisabledStatus(serial string) pkisvc.OCSPStatus {
	reason := "ocspDisabled"
	return pkisvc.OCSPStatus{Status: "unknown", Valid: false, Reason: &reason, SerialNumber: serial, CheckedAt: time.Now().Local().Format("2006-01-02 15:04:05")}
}
func (r *Router) updateWorkflow(w http.ResponseWriter, q *http.Request) {
	id := q.PathValue("id")
	if id == "" {
		write(w, 404, map[string]string{"message": "审批记录不存在"})
		return
	}
	var body struct {
		Status       string `json:"status"`
		RejectReason string `json:"reject_reason"`
	}
	if err := decode(q, &body); err != nil {
		write(w, 400, map[string]string{"message": "请求格式不正确"})
		return
	}
	if err := r.pki.Approve(q.Context(), id, body.Status, body.RejectReason, current(q), clientIP(q)); err != nil {
		status := 400
		if errors.Is(err, repository.ErrNotFound) {
			status = 404
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	write(w, 200, map[string]string{"status": "ok"})
}
