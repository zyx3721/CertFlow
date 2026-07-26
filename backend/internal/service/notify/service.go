package notify

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"certflow/backend/internal/repository"
	"certflow/backend/internal/security"
)

type Service struct {
	store *repository.Store
	box   *security.Cryptobox
}

type EmailConfig struct {
	SMTPHost          string `json:"smtpHost"`
	SMTPPort          int    `json:"smtpPort"`
	Username          string `json:"username"`
	Password          string `json:"-"`
	From              string `json:"from"`
	FromName          string `json:"fromName"`
	UseTLS            bool   `json:"useTLS"`
	StartTLS          bool   `json:"startTLS"`
	AllowInsecureAuth bool   `json:"allowInsecureAuth"`
}

func New(store *repository.Store, box *security.Cryptobox) *Service {
	return &Service{store: store, box: box}
}

func (s *Service) Setting(ctx context.Context) (repository.NotificationChannelSetting, error) {
	item, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil {
		return item, err
	}
	item.Config["hasPassword"] = item.SecretCiphertext != ""
	return item, nil
}

func (s *Service) SaveSetting(ctx context.Context, enabled bool, clear bool, values map[string]any) (repository.NotificationChannelSetting, error) {
	current, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return current, err
	}
	if clear {
		current.SecretCiphertext = ""
	}
	values = sanitizeEmailSettingValues(values)
	password, _ := values["password"].(string)
	password = strings.TrimSpace(password)
	delete(values, "password")
	if password != "" {
		current.SecretCiphertext, err = s.box.Seal(password)
		if err != nil {
			return current, err
		}
	}
	current.ID = "email"
	current.Name = "邮件"
	current.PasswordResetEnabled = enabled
	current.Config = values
	if enabled {
		if err := s.validateRequiredEmailSetting(current); err != nil {
			return current, err
		}
		if err := applyEmailTransportSettings(current.Config); err != nil {
			return current, err
		}
		if _, err := s.config(current); err != nil {
			return current, err
		}
	}
	if err := s.store.SaveNotificationChannelSetting(ctx, current); err != nil {
		return current, err
	}
	return s.Setting(ctx)
}

func (s *Service) Test(ctx context.Context, to string) error {
	setting, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil {
		return err
	}
	if !setting.PasswordResetEnabled {
		return errors.New("找回密码邮件未启用")
	}
	to = strings.TrimSpace(to)
	if _, err := mail.ParseAddress(to); err != nil {
		return errors.New("收件人格式不正确")
	}
	cfg, err := s.config(setting)
	if err != nil {
		return err
	}
	return sendSMTP(ctx, cfg, to, buildTestMessage(cfg, to, s.siteName(ctx)))
}

func (s *Service) SendPasswordReset(ctx context.Context, to string, username string, code string, expiresAt time.Time, requestIP string) error {
	return s.send(ctx, to, "CertFlow 密码找回验证码", buildPasswordResetEmail(username, code, expiresAt, requestIP))
}

func (s *Service) send(ctx context.Context, to string, subject string, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return errors.New("收件人格式不正确")
	}
	setting, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil {
		return err
	}
	if !setting.PasswordResetEnabled && subject != "CertFlow 测试邮件" {
		return errors.New("找回密码邮件未启用")
	}
	cfg, err := s.config(setting)
	if err != nil {
		return err
	}
	message := buildHTMLMessage(cfg, to, subject, body)
	return sendSMTP(ctx, cfg, to, message)
}

func (s *Service) siteName(ctx context.Context) string {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return "CertFlow"
	}
	name := stringConfig(settings, "siteName")
	if name == "" {
		return "CertFlow"
	}
	return name
}

func (s *Service) config(setting repository.NotificationChannelSetting) (EmailConfig, error) {
	values := setting.Config
	cfg := EmailConfig{
		SMTPHost:          stringConfig(values, "smtpHost"),
		SMTPPort:          intConfig(values, "smtpPort"),
		Username:          stringConfig(values, "username"),
		From:              stringConfig(values, "from"),
		FromName:          stringConfig(values, "fromName"),
		UseTLS:            boolConfig(values, "useTLS"),
		StartTLS:          boolConfig(values, "startTLS"),
		AllowInsecureAuth: boolConfig(values, "allowInsecureAuth"),
	}
	if setting.SecretCiphertext != "" {
		password, err := s.box.Open(setting.SecretCiphertext)
		if err != nil {
			return cfg, err
		}
		cfg.Password = password
	}
	if cfg.SMTPHost == "" {
		return cfg, errors.New("SMTP 主机不能为空")
	}
	if !emailConfigValuePresent(values["smtpPort"]) {
		return cfg, errors.New("SMTP 端口不能为空")
	}
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return cfg, errors.New("SMTP 端口需为 1 到 65535 之间的整数")
	}
	if cfg.Username == "" {
		return cfg, errors.New("用户名不能为空")
	}
	if cfg.Password == "" {
		return cfg, errors.New("密码不能为空")
	}
	if cfg.From == "" {
		return cfg, errors.New("发件人不能为空")
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return cfg, errors.New("发件人格式不正确")
	}
	cfg.From = from.Address
	if cfg.UseTLS && cfg.StartTLS {
		return cfg, errors.New("TLS 与 STARTTLS 不能同时启用")
	}
	return cfg, nil
}

func (s *Service) validateRequiredEmailSetting(setting repository.NotificationChannelSetting) error {
	values := setting.Config
	if stringConfig(values, "smtpHost") == "" {
		return errors.New("SMTP 主机不能为空")
	}
	if !emailConfigValuePresent(values["smtpPort"]) {
		return errors.New("SMTP 端口不能为空")
	}
	if stringConfig(values, "username") == "" {
		return errors.New("用户名不能为空")
	}
	if setting.SecretCiphertext == "" {
		return errors.New("密码不能为空")
	}
	if stringConfig(values, "from") == "" {
		return errors.New("发件人不能为空")
	}
	return nil
}

func emailConfigValuePresent(value any) bool {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return value != nil
}

func sanitizeEmailSettingValues(values map[string]any) map[string]any {
	cleaned := make(map[string]any, len(values))
	for key, value := range values {
		switch key {
		case "hasPassword", "passwordConfigured", "fromAddress", "useTls", "startTls", "insecureSkipVerify", "timeoutSeconds", "testRecipient", "to":
			continue
		}
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			cleaned[key] = text
			continue
		}
		if value != nil {
			cleaned[key] = value
		}
	}
	if _, ok := cleaned["from"]; !ok {
		if value, exists := values["fromAddress"].(string); exists {
			if value = strings.TrimSpace(value); value != "" {
				cleaned["from"] = value
			}
		}
	}
	if _, ok := cleaned["useTLS"]; !ok {
		if value, exists := values["useTls"].(bool); exists {
			cleaned["useTLS"] = value
		}
	}
	if _, ok := cleaned["startTLS"]; !ok {
		if value, exists := values["startTls"].(bool); exists {
			cleaned["startTLS"] = value
		}
	}
	return cleaned
}

func applyEmailTransportSettings(values map[string]any) error {
	useTLS := boolConfig(values, "useTLS")
	startTLS := boolConfig(values, "startTLS")
	if useTLS && startTLS {
		return errors.New("TLS 与 STARTTLS 不能同时启用")
	}
	if useTLS {
		values["smtpPort"] = 465
		values["allowInsecureAuth"] = false
		return nil
	}
	if startTLS {
		values["smtpPort"] = 587
		values["allowInsecureAuth"] = false
		return nil
	}
	port := intConfig(values, "smtpPort")
	if port < 1 || port > 65535 {
		return errors.New("SMTP 端口需为 1 到 65535 之间的整数")
	}
	values["smtpPort"] = port
	return nil
}

func sendSMTP(ctx context.Context, cfg EmailConfig, to string, message []byte) error {
	address := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var client *smtp.Client
	var err error
	if cfg.UseTLS {
		var connection net.Conn
		connection, err = (&tls.Dialer{NetDialer: dialer, Config: &tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12}}).DialContext(ctx, "tcp", address)
		if err == nil {
			client, err = smtp.NewClient(connection, cfg.SMTPHost)
		}
	} else {
		var connection net.Conn
		connection, err = dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			_ = connection.SetDeadline(time.Now().Add(10 * time.Second))
			client, err = smtp.NewClient(connection, cfg.SMTPHost)
		}
		if err == nil && cfg.StartTLS {
			err = client.StartTLS(&tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12})
		}
	}
	if err != nil {
		return errors.New("SMTP 连接失败")
	}
	defer client.Close()
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	if !cfg.UseTLS && !cfg.StartTLS && cfg.AllowInsecureAuth {
		auth = plainInsecureAuthPayload(cfg.Username, cfg.Password)
	}
	if err := client.Auth(auth); err != nil {
		return errors.New("SMTP 认证失败")
	}
	if err := client.Mail(cfg.From); err != nil {
		return errors.New("SMTP 发件人被拒绝")
	}
	if err := client.Rcpt(to); err != nil {
		return errors.New("SMTP 收件人被拒绝")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("SMTP 邮件内容发送失败")
	}
	if _, err = writer.Write(message); err != nil {
		_ = writer.Close()
		return errors.New("SMTP 邮件内容发送失败")
	}
	if err = writer.Close(); err != nil {
		return errors.New("SMTP 邮件发送失败")
	}
	return client.Quit()
}

type plainInsecureAuth string

func plainInsecureAuthPayload(username string, password string) smtp.Auth {
	return plainInsecureAuth("\x00" + username + "\x00" + password)
}

func (a plainInsecureAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte(a), nil
}

func (a plainInsecureAuth) Next([]byte, bool) ([]byte, error) {
	return nil, nil
}

func buildHTMLMessage(cfg EmailConfig, to string, subject string, body string) []byte {
	fromName := strings.TrimSpace(cfg.FromName)
	if fromName == "" {
		fromName = "CertFlow"
	}
	from := (&mail.Address{Name: fromName, Address: cfg.From}).String()
	return []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", from, to, subject, body))
}

func buildPasswordResetEmail(username string, code string, expiresAt time.Time, requestIP string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "当前账号"
	}
	requestIP = strings.TrimSpace(requestIP)
	if requestIP == "" {
		requestIP = "未知"
	}
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#f5f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f5f7fb;padding:32px 12px;"><tr><td align="center">
<table role="presentation" width="560" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #e4e9f2;border-radius:14px;overflow:hidden;">
<tr><td style="padding:28px 32px 18px;background:#0e7490;color:#ffffff;"><div style="font-size:20px;font-weight:700;">CertFlow 密码找回</div><div style="margin-top:8px;font-size:13px;opacity:.86;">请使用以下验证码完成密码重置</div></td></tr>
<tr><td style="padding:30px 32px;"><div style="font-size:14px;color:#526071;">账号</div><div style="margin-top:6px;font-size:18px;font-weight:700;color:#172033;">%s</div><div style="margin-top:24px;padding:18px 20px;border-radius:12px;background:#ecfeff;border:1px solid #a5f3fc;text-align:center;"><div style="font-size:13px;color:#0e7490;">验证码</div><div style="margin-top:8px;font-size:34px;letter-spacing:8px;font-weight:800;color:#155e75;">%s</div></div><div style="margin-top:22px;font-size:14px;line-height:1.8;color:#526071;">有效期至：<strong style="color:#172033;">%s</strong><br>请求来源：<strong style="color:#172033;">%s</strong></div><div style="margin-top:24px;padding:14px 16px;border-radius:10px;background:#fff7ed;border:1px solid #fed7aa;color:#9a3412;font-size:13px;line-height:1.7;">如果不是您本人操作，请忽略本邮件并检查平台账号安全。</div></td></tr>
</table></td></tr></table></body></html>`, html.EscapeString(username), html.EscapeString(code), html.EscapeString(expiresAt.Local().Format("2006-01-02 15:04:05")), html.EscapeString(requestIP))
}

func buildTestMessage(cfg EmailConfig, to string, siteName string) []byte {
	siteName = strings.TrimSpace(siteName)
	if siteName == "" {
		siteName = "CertFlow"
	}
	return buildHTMLMessage(cfg, to, siteName+" 平台邮件配置测试", notificationEmailBody("这是一封 "+siteName+" 邮件配置测试邮件。", notificationToneNormal))
}

func stringConfig(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func intConfig(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func boolConfig(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}
