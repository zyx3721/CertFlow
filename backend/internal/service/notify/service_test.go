package notify

import (
	"strings"
	"testing"
	"time"

	"certflow/backend/internal/repository"
)

func TestApplyEmailTransportSettings(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]any
		wantPort  int
		wantTLS   bool
		wantStart bool
		wantErr   bool
	}{
		{
			name:      "implicit TLS uses port 465",
			values:    map[string]any{"useTLS": true, "startTLS": false, "allowInsecureAuth": true},
			wantPort:  465,
			wantTLS:   true,
			wantStart: false,
		},
		{
			name:      "STARTTLS uses port 587",
			values:    map[string]any{"useTLS": false, "startTLS": true, "allowInsecureAuth": true},
			wantPort:  587,
			wantTLS:   false,
			wantStart: true,
		},
		{
			name:    "conflicting TLS modes fail",
			values:  map[string]any{"useTLS": true, "startTLS": true},
			wantErr: true,
		},
		{
			name:    "plain SMTP requires a valid port",
			values:  map[string]any{"useTLS": false, "startTLS": false, "smtpPort": "invalid"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := applyEmailTransportSettings(test.values)
			if (err != nil) != test.wantErr {
				t.Fatalf("applyEmailTransportSettings() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if test.values["smtpPort"] != test.wantPort || boolConfig(test.values, "useTLS") != test.wantTLS || boolConfig(test.values, "startTLS") != test.wantStart {
				t.Fatalf("unexpected transport settings: %#v", test.values)
			}
			if boolConfig(test.values, "allowInsecureAuth") {
				t.Fatalf("secure transport must disable insecure authentication: %#v", test.values)
			}
		})
	}
}

func TestSanitizeEmailSettingValues(t *testing.T) {
	values := sanitizeEmailSettingValues(map[string]any{
		"fromAddress":        "  sender@example.com  ",
		"useTls":             true,
		"startTls":           false,
		"insecureSkipVerify": true,
		"timeoutSeconds":     8,
		"hasPassword":        true,
		"smtpHost":           "  smtp.example.com  ",
		"allowInsecureAuth":  false,
		"passwordConfigured": true,
	})
	if values["from"] != "sender@example.com" || values["smtpHost"] != "smtp.example.com" || !boolConfig(values, "useTLS") {
		t.Fatalf("unexpected normalized values: %#v", values)
	}
	for _, key := range []string{"fromAddress", "useTls", "startTls", "insecureSkipVerify", "timeoutSeconds", "hasPassword", "passwordConfigured"} {
		if _, ok := values[key]; ok {
			t.Fatalf("legacy or marker field %q must be removed: %#v", key, values)
		}
	}
}

func TestValidateRequiredEmailSettingChecksMandatoryFieldsFirst(t *testing.T) {
	service := &Service{}
	setting := repository.NotificationChannelSetting{
		SecretCiphertext: "configured",
		Config: map[string]any{
			"smtpHost": "",
			"smtpPort": "invalid",
			"username": "operator",
			"from":     "sender@example.com",
		},
	}
	if err := service.validateRequiredEmailSetting(setting); err == nil || err.Error() != "SMTP 主机不能为空" {
		t.Fatalf("expected SMTP host error before transport checks, got %v", err)
	}
	setting.Config["smtpHost"] = "smtp.example.com"
	setting.Config["username"] = ""
	if err := service.validateRequiredEmailSetting(setting); err == nil || err.Error() != "用户名不能为空" {
		t.Fatalf("expected username error before transport checks, got %v", err)
	}
}

func TestBuildTestMessageUsesGreenHTMLTemplate(t *testing.T) {
	message := string(buildTestMessage(EmailConfig{
		From:     "sender@example.com",
		FromName: "CertFlow",
	}, "admin@example.com", "Example PKI"))
	for _, expected := range []string{
		"From: \"CertFlow\" <sender@example.com>",
		"To: admin@example.com",
		"Subject: Example PKI 平台邮件配置测试",
		"Content-Type: text/html; charset=UTF-8",
		"color:#2BDE3F",
		"这是一封 Example PKI 邮件配置测试邮件。",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("test message does not contain %q: %s", expected, message)
		}
	}
}

func TestValidateChannelConfigUsesFieldLabels(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		config map[string]any
		secret string
		want   string
	}{
		{
			name: "webhook URL",
			id:   "webhook",
			want: "Webhook URL 不能为空",
		},
		{
			name: "dingtalk signing secret",
			id:   "dingtalk",
			config: map[string]any{
				"webhookUrl": "https://oapi.dingtalk.com/robot/send?access_token=test",
			},
			want: "加签密钥不能为空",
		},
		{
			name: "lark application secret before receive fields",
			id:   "lark_app",
			config: map[string]any{
				"appId": "cli_example",
			},
			want: "App Secret 不能为空",
		},
		{
			name: "lark receive ID type",
			id:   "lark_app",
			config: map[string]any{
				"appId":     "cli_example",
				"receiveId": "oc_example",
			},
			secret: "configured",
			want:   "接收 ID 类型不能为空",
		},
		{
			name: "dingtalk application secret",
			id:   "dingtalk_app",
			config: map[string]any{
				"appKey":     "ding_example",
				"agentId":    "1000001",
				"useridList": "user_001",
			},
			want: "AppSecret 不能为空",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateChannelConfig(test.id, test.config, test.secret)
			if err == nil || err.Error() != test.want {
				t.Fatalf("validateChannelConfig() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLarkApprovalCardUsesGreenHeader(t *testing.T) {
	payload := larkApprovalCard("CertFlow 平台审批通知测试", "这是一条审批通知测试消息", notificationToneNormal)
	if payload["msg_type"] != "interactive" {
		t.Fatalf("msg_type = %v, want interactive", payload["msg_type"])
	}
	card, ok := payload["card"].(map[string]any)
	if !ok {
		t.Fatalf("card payload missing: %#v", payload)
	}
	header, ok := card["header"].(map[string]any)
	if !ok || header["template"] != "green" {
		t.Fatalf("card header = %#v, want green", header)
	}
}

func TestLarkApprovalCardUsesWarningTone(t *testing.T) {
	payload := larkApprovalCard("CertFlow 平台证书续期通知", "自动续期失败", notificationToneWarning)
	card := payload["card"].(map[string]any)
	header := card["header"].(map[string]any)
	if header["template"] != "red" {
		t.Fatalf("card header = %#v, want red", header)
	}
	elements := card["elements"].([]map[string]any)
	text := elements[0]["text"].(map[string]string)["content"]
	if text != "<font color='orange'>自动续期失败</font>" {
		t.Fatalf("card body = %q, want orange text", text)
	}
}

func TestDingTalkApprovalMarkdownUsesHeadingsAndLineBreaks(t *testing.T) {
	message := dingTalkApprovalMarkdown("CertFlow 平台审批通知测试", "第一行\n第二行", notificationToneNormal)
	if message != "## CertFlow 平台审批通知测试  \n<font color=\"green\">第一行  \n第二行  </font>  " {
		t.Fatalf("unexpected DingTalk markdown: %q", message)
	}
}

func TestNotificationMessageStyles(t *testing.T) {
	if message := wechatNotificationMessage("标题", "内容", notificationToneNormal); !strings.Contains(message, `<font color="info">内容</font>`) {
		t.Fatalf("WeChat normal message = %q", message)
	}
	if message := wechatNotificationMessage("标题", "内容", notificationToneWarning); !strings.Contains(message, `<font color="warning">内容</font>`) {
		t.Fatalf("WeChat warning message = %q", message)
	}
	if message := dingTalkApprovalMarkdown("标题", "内容", notificationToneWarning); !strings.Contains(message, `<font color="orange">内容  </font>`) {
		t.Fatalf("DingTalk warning message = %q", message)
	}
	if body := notificationEmailBody("内容", notificationToneNormal); !strings.Contains(body, "#2BDE3F") {
		t.Fatalf("email normal body = %q", body)
	}
	if body := notificationEmailBody("内容", notificationToneWarning); !strings.Contains(body, "#FFC007") {
		t.Fatalf("email warning body = %q", body)
	}
}

func TestCertificateExpiryMessage(t *testing.T) {
	notAfter := time.Date(2026, 8, 1, 9, 30, 0, 0, time.Local)
	message := certificateExpiryMessage("example.com", notAfter, 7)
	if message != "证书 example.com 将于 2026.08.01 09:30:00 到期，剩余 7 天，请及时续期" {
		t.Fatalf("unexpected certificate expiry message: %q", message)
	}
}

func TestLarkReceiveIDTypeValidation(t *testing.T) {
	if !isLarkReceiveIDType("chat_id") || !isLarkReceiveIDType("email") {
		t.Fatal("supported receive ID types must pass validation")
	}
	if isLarkReceiveIDType("member_id") {
		t.Fatal("unsupported receive ID type must fail validation")
	}
}

func TestNormalizeWebhookHeaders(t *testing.T) {
	headers, err := normalizeWebhookHeaders(`{"Authorization":"Bearer token","X-Source":"certflow"}`)
	if err != nil {
		t.Fatalf("normalizeWebhookHeaders() error = %v", err)
	}
	if headers["Authorization"] != "Bearer token" || headers["X-Source"] != "certflow" {
		t.Fatalf("unexpected headers: %#v", headers)
	}
	if _, err := normalizeWebhookHeaders(`{"X-Retry":1}`); err == nil {
		t.Fatal("non-string header values must be rejected")
	}
	if _, err := normalizeWebhookHeaders(`["X-Test"]`); err == nil {
		t.Fatal("non-object headers must be rejected")
	}
}

func TestBuildPasswordResetEmailUsesCardTemplate(t *testing.T) {
	expiresAt := time.Date(2026, 7, 22, 1, 3, 10, 0, time.Local)
	body := buildPasswordResetEmail("admin", "851577", expiresAt, "192.0.2.10")
	for _, expected := range []string{
		"CertFlow 密码找回",
		"请使用以下验证码完成密码重置",
		">admin<",
		">851577<",
		"2026-07-22 01:03:10",
		"192.0.2.10",
		"如果不是您本人操作，请忽略本邮件并检查平台账号安全",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("password reset template does not contain %q: %s", expected, body)
		}
	}
}
