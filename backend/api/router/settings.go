package router

import (
	"net/http"
	"strings"

	"github.com/robfig/cron/v3"
)

type pkiSettings struct {
	SiteName                         string  `json:"siteName"`
	LoginName                        string  `json:"loginName"`
	AppName                          string  `json:"appName"`
	AppSubtitle                      string  `json:"appSubtitle"`
	IconData                         string  `json:"iconData"`
	CRLEnabled                       bool    `json:"crlEnabled"`
	CRLURL                           string  `json:"crlUrl"`
	OCSPEnabled                      bool    `json:"ocspEnabled"`
	OCSPURL                          string  `json:"ocspUrl"`
	CRLIntervalHours                 int     `json:"crlIntervalHours"`
	RenewDays                        int     `json:"renewDays"`
	AutoRenew                        bool    `json:"autoRenew"`
	ExpiryNotificationDays           int     `json:"expiryNotificationDays"`
	ExpiryNotificationCron           string  `json:"expiryNotificationCron"`
	ResetCodeTTLMinutes              int     `json:"resetCodeTtlMinutes"`
	ResetCaptchaTTLMinutes           int     `json:"resetCaptchaTtlMinutes"`
	PasswordResetSendCooldownMinutes float64 `json:"passwordResetSendCooldownMinutes"`
	PasswordResetRateLimitMinutes    int     `json:"passwordResetRateLimitMinutes"`
}

func (r *Router) publicSettings(w http.ResponseWriter, q *http.Request) {
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusOK, publicSettingsPayload(nil))
		return
	}
	write(w, http.StatusOK, publicSettingsPayload(settings))
}

func publicSettingsPayload(settings map[string]any) map[string]any {
	return map[string]any{
		"siteName":               settingText(settings, "siteName", "CertFlow"),
		"loginName":              settingText(settings, "loginName", "CertFlow"),
		"appName":                settingText(settings, "appName", "CertFlow"),
		"appSubtitle":            settingText(settings, "appSubtitle", "PKI Control Plane"),
		"iconData":               settingText(settings, "iconData", "/favicon.svg"),
		"expiryNotificationDays": expiryNotificationDays(settings),
	}
}

func settingText(settings map[string]any, key string, fallback string) string {
	if value, ok := settings[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func expiryNotificationDays(settings map[string]any) int {
	if value, ok := settings["expiryNotificationDays"].(float64); ok && value >= 1 && value <= 365 {
		return int(value)
	}
	return 15
}

func (r *Router) getSettings(w http.ResponseWriter, q *http.Request) {
	settings, err := r.store.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取设置失败"})
		return
	}
	write(w, http.StatusOK, settings)
}

func (r *Router) saveSettings(w http.ResponseWriter, q *http.Request) {
	var settings pkiSettings
	if err := decode(q, &settings); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	settings.SiteName = strings.TrimSpace(settings.SiteName)
	settings.LoginName = strings.TrimSpace(settings.LoginName)
	settings.AppName = strings.TrimSpace(settings.AppName)
	settings.AppSubtitle = strings.TrimSpace(settings.AppSubtitle)
	settings.IconData = strings.TrimSpace(settings.IconData)
	settings.CRLURL = strings.TrimSpace(settings.CRLURL)
	settings.OCSPURL = strings.TrimSpace(settings.OCSPURL)
	settings.ExpiryNotificationCron = strings.TrimSpace(settings.ExpiryNotificationCron)
	if settings.SiteName == "" || settings.LoginName == "" || settings.AppName == "" || settings.CRLIntervalHours < 1 || settings.CRLIntervalHours > 720 || settings.RenewDays < 1 || settings.RenewDays > 365 || settings.ExpiryNotificationDays < 1 || settings.ExpiryNotificationDays > 365 {
		write(w, http.StatusBadRequest, map[string]string{"message": "基础配置、CRL 更新间隔、续期天数或到期提醒阈值不合法"})
		return
	}
	if _, err := cron.ParseStandard(settings.ExpiryNotificationCron); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "定时扫描计划格式不正确"})
		return
	}
	if len(settings.IconData) > 350*1024 || (settings.IconData != "" && !strings.HasPrefix(settings.IconData, "/") && !strings.HasPrefix(settings.IconData, "data:image/")) {
		write(w, http.StatusBadRequest, map[string]string{"message": "品牌图标格式或大小不合法"})
		return
	}
	if settings.ResetCodeTTLMinutes < 1 || settings.ResetCodeTTLMinutes > 60 || settings.ResetCaptchaTTLMinutes < 1 || settings.ResetCaptchaTTLMinutes > 10 || settings.PasswordResetSendCooldownMinutes < 0.5 || settings.PasswordResetSendCooldownMinutes > 10 || settings.PasswordResetRateLimitMinutes < 1 || settings.PasswordResetRateLimitMinutes > 60 {
		write(w, http.StatusBadRequest, map[string]string{"message": "找回密码安全时效配置不合法"})
		return
	}
	if settings.CRLEnabled && settings.CRLURL == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "启用 CRL 时必须配置分发地址"})
		return
	}
	if settings.OCSPEnabled && settings.OCSPURL == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "启用 OCSP 时必须配置响应器地址"})
		return
	}
	settingsMap := map[string]any{
		"siteName":                         settings.SiteName,
		"loginName":                        settings.LoginName,
		"appName":                          settings.AppName,
		"appSubtitle":                      settings.AppSubtitle,
		"iconData":                         settings.IconData,
		"crlEnabled":                       settings.CRLEnabled,
		"crlUrl":                           settings.CRLURL,
		"ocspEnabled":                      settings.OCSPEnabled,
		"ocspUrl":                          settings.OCSPURL,
		"crlIntervalHours":                 settings.CRLIntervalHours,
		"renewDays":                        settings.RenewDays,
		"autoRenew":                        settings.AutoRenew,
		"expiryNotificationDays":           settings.ExpiryNotificationDays,
		"expiryNotificationCron":           settings.ExpiryNotificationCron,
		"resetCodeTtlMinutes":              settings.ResetCodeTTLMinutes,
		"resetCaptchaTtlMinutes":           settings.ResetCaptchaTTLMinutes,
		"passwordResetSendCooldownMinutes": settings.PasswordResetSendCooldownMinutes,
		"passwordResetRateLimitMinutes":    settings.PasswordResetRateLimitMinutes,
	}

	if err := r.store.SaveSettings(q.Context(), settingsMap); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "保存设置失败"})
		return
	}

	r.store.Audit(q.Context(), current(q), "保存系统设置", "system_settings", "settings", "success", clientIP(q), "系统设置已更新")
	write(w, http.StatusOK, settingsMap)
}
