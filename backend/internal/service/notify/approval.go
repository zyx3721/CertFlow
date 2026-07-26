package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
)

var channelNames = map[string]string{
	"webhook": "Webhook", "email": "邮件", "lark": "飞书机器人", "lark_app": "飞书应用",
	"wechat": "企业微信机器人", "wechat_app": "企业微信应用", "dingtalk": "钉钉机器人", "dingtalk_app": "钉钉应用",
}

func (s *Service) Settings(ctx context.Context) ([]repository.NotificationChannelSetting, error) {
	items, err := s.store.ListNotificationChannelSettings(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		if items[index].ID == "email" {
			items[index].Config["hasPassword"] = items[index].SecretCiphertext != ""
		} else if items[index].SecretCiphertext != "" {
			items[index].Config["hasSecret"] = true
		}
	}
	return items, nil
}

func (s *Service) SaveChannel(ctx context.Context, id string, passwordResetEnabled, approvalEnabled, clear bool, values map[string]any) (repository.NotificationChannelSetting, error) {
	if _, ok := channelNames[id]; !ok {
		return repository.NotificationChannelSetting{}, errors.New("通知媒介不存在")
	}
	if id == "email" {
		return s.saveEmailChannel(ctx, passwordResetEnabled, approvalEnabled, clear, values)
	}
	current, err := s.store.NotificationChannelSetting(ctx, id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return current, err
	}
	wasApprovalEnabled := current.ApprovalEnabled
	if clear {
		current.SecretCiphertext = ""
	}
	config, secret, err := sanitizeChannelConfig(id, values, current.SecretCiphertext, s.box)
	if err != nil {
		return current, err
	}
	current.ID, current.Name, current.PasswordResetEnabled, current.ApprovalEnabled = id, channelNames[id], false, approvalEnabled
	current.Config, current.SecretCiphertext = config, secret
	if approvalEnabled {
		if err := validateChannelConfig(id, config, secret); err != nil {
			return current, err
		}
	}
	if err := s.store.SaveNotificationChannelSetting(ctx, current); err != nil {
		return current, err
	}
	if wasApprovalEnabled && !approvalEnabled {
		if err := s.store.ClearCertificateExpiryNotificationsForChannel(ctx, id); err != nil {
			return current, err
		}
	}
	return s.channelSetting(ctx, id)
}

func (s *Service) channelSetting(ctx context.Context, id string) (repository.NotificationChannelSetting, error) {
	item, err := s.store.NotificationChannelSetting(ctx, id)
	if err != nil {
		return item, err
	}
	if id == "email" {
		item.Config["hasPassword"] = item.SecretCiphertext != ""
	} else if item.SecretCiphertext != "" {
		item.Config["hasSecret"] = true
	}
	return item, nil
}

func (s *Service) saveEmailChannel(ctx context.Context, passwordResetEnabled, approvalEnabled, clear bool, values map[string]any) (repository.NotificationChannelSetting, error) {
	current, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return current, err
	}
	wasApprovalEnabled := current.ApprovalEnabled
	if clear {
		current.SecretCiphertext = ""
	}
	values = sanitizeEmailSettingValues(values)
	password := strings.TrimSpace(stringConfig(values, "password"))
	delete(values, "password")
	if password != "" {
		current.SecretCiphertext, err = s.box.Seal(password)
		if err != nil {
			return current, err
		}
	}
	current.ID, current.Name, current.PasswordResetEnabled, current.ApprovalEnabled, current.Config = "email", "邮件", passwordResetEnabled, approvalEnabled, values
	if passwordResetEnabled || approvalEnabled {
		if err := s.validateRequiredEmailSetting(current); err != nil {
			return current, err
		}
		if err := applyEmailTransportSettings(current.Config); err != nil {
			return current, err
		}
	}
	if err := s.store.SaveNotificationChannelSetting(ctx, current); err != nil {
		return current, err
	}
	if wasApprovalEnabled && !approvalEnabled {
		if err := s.store.ClearCertificateExpiryNotificationsForChannel(ctx, "email"); err != nil {
			return current, err
		}
	}
	return s.channelSetting(ctx, "email")
}

func (s *Service) NotifyApproval(ctx context.Context, applicant, commonName string, createdAt time.Time) {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return
	}
	brand := strings.TrimSpace(stringConfig(settings, "siteName"))
	if brand == "" {
		brand = "CertFlow"
	}
	message := fmt.Sprintf("用户 %s 于 %s 申请了 %s 证书，请前往平台进行审批", applicant, createdAt.Format("2006.01.02 15:04:05"), commonName)
	title := brand + " 平台审批通知"
	channels, err := s.store.ListNotificationChannelSettings(ctx)
	if err != nil {
		return
	}
	for _, channel := range channels {
		if !channel.ApprovalEnabled {
			continue
		}
		if channel.ID == "email" {
			_ = s.sendApprovalEmail(ctx, channel, title, message, notificationToneNormal)
			continue
		}
		_ = s.sendApprovalChannel(ctx, channel, title, message, notificationToneNormal)
	}
}

func (s *Service) TestChannel(ctx context.Context, id, to string) error {
	channel, err := s.store.NotificationChannelSetting(ctx, id)
	if err != nil {
		return err
	}
	if id == "email" {
		if !channel.PasswordResetEnabled && !channel.ApprovalEnabled {
			return errors.New("邮件媒介未启用")
		}
		to = strings.TrimSpace(to)
		if _, err := mail.ParseAddress(to); err != nil {
			return errors.New("收件人格式不正确")
		}
		cfg, err := s.config(channel)
		if err != nil {
			return err
		}
		return sendSMTP(ctx, cfg, to, buildTestMessage(cfg, to, s.siteName(ctx)))
	}
	if !channel.ApprovalEnabled {
		return errors.New("通知审批未启用")
	}
	return s.sendApprovalChannel(ctx, channel, s.siteName(ctx)+" 平台通知测试", "这是一条通知测试消息", notificationToneNormal)
}

func (s *Service) sendApprovalEmail(ctx context.Context, channel repository.NotificationChannelSetting, title, message string, tone notificationTone) error {
	return s.sendEmailToUsersWithPermission(ctx, channel, title, message, "workflows.approve", tone)
}

func (s *Service) sendExpiryEmail(ctx context.Context, channel repository.NotificationChannelSetting, title, message string, tone notificationTone) error {
	return s.sendEmailToUsersWithPermission(ctx, channel, title, message, "certificates.read", tone)
}

func (s *Service) sendEmailToUsersWithPermission(ctx context.Context, channel repository.NotificationChannelSetting, title, message, permission string, tone notificationTone) error {
	cfg, err := s.config(channel)
	if err != nil {
		return err
	}
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return err
	}
	var deliveryError error
	for _, user := range users {
		if user.Disabled || !hasPermission(user, permission) {
			continue
		}
		if _, err := mail.ParseAddress(user.Email); err != nil {
			continue
		}
		if err := sendSMTP(ctx, cfg, user.Email, buildHTMLMessage(cfg, user.Email, title, notificationEmailBody(message, tone))); err != nil {
			deliveryError = err
		}
	}
	return deliveryError
}

func hasPermission(user domain.User, permission string) bool {
	for _, item := range user.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func (s *Service) sendApprovalChannel(ctx context.Context, channel repository.NotificationChannelSetting, title, message string, tone notificationTone) error {
	config := channel.Config
	secrets, _ := openChannelSecrets(channel.SecretCiphertext, s.box)
	value := func(key string) string {
		if text := strings.TrimSpace(stringConfig(config, key)); text != "" {
			return text
		}
		return secrets[key]
	}
	switch channel.ID {
	case "webhook":
		method := strings.ToUpper(value("method"))
		if method == "" {
			method = http.MethodPost
		}
		return postJSON(ctx, method, value("url"), map[string]string{"title": title, "message": message}, webhookHeaders(config))
	case "lark":
		payload := larkApprovalCard(title, message, tone)
		if secret := value("secret"); secret != "" {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			payload["timestamp"] = timestamp
			payload["sign"] = signLark(timestamp, secret)
		}
		return sendLarkRobot(ctx, value("webhookUrl"), payload)
	case "wechat":
		return sendWechatRobot(ctx, value("webhookUrl"), title, message, tone)
	case "dingtalk":
		webhookURL, err := signDingTalkURL(value("webhookUrl"), value("secret"))
		if err != nil {
			return err
		}
		return sendDingTalkRobot(ctx, webhookURL, title, message, tone)
	case "lark_app":
		return sendLarkAppApproval(ctx, value, title, message, tone)
	case "wechat_app":
		return sendWechatAppApproval(ctx, value, title, message, tone)
	case "dingtalk_app":
		return sendDingTalkAppApproval(ctx, value, title, message, tone)
	}
	return nil
}

func larkApprovalCard(title, message string, tone notificationTone) map[string]any {
	return map[string]any{
		"msg_type": "interactive",
		"card":     larkApprovalCardBody(title, message, tone),
	}
}

func larkApprovalCardBody(title, message string, tone notificationTone) map[string]any {
	return map[string]any{
		"config": map[string]bool{"wide_screen_mode": true},
		"header": map[string]any{
			"template": larkHeaderTemplate(tone),
			"title":    map[string]string{"tag": "plain_text", "content": title},
		},
		"elements": []map[string]any{
			{"tag": "div", "text": map[string]string{"tag": "lark_md", "content": larkNotificationMessage(message, tone)}},
		},
	}
}

func mustJSONString(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func signLark(timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(timestamp+"\n"+secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func signDingTalkURL(rawURL, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", requiredNotificationFieldError(notificationFieldLabel("dingtalk", "secret"))
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func firstNotificationError(message, fallback string) string {
	if message = strings.TrimSpace(message); message != "" {
		return message
	}
	return fallback
}

func sendLarkAppApproval(ctx context.Context, value func(string) string, title, message string, tone notificationTone) error {
	var token struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Token string `json:"tenant_access_token"`
	}
	if err := postJSONDecode(ctx, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", map[string]string{"app_id": value("appId"), "app_secret": value("appSecret")}, nil, &token); err != nil {
		return err
	}
	if token.Code != 0 || token.Token == "" {
		return errors.New("飞书获取访问令牌失败")
	}
	payload := map[string]any{"receive_id": value("receiveId"), "msg_type": "interactive", "content": mustJSONString(larkApprovalCardBody(title, message, tone))}
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := postJSONDecode(ctx, "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+url.QueryEscape(value("receiveIdType")), payload, map[string]string{"Authorization": "Bearer " + token.Token}, &response); err != nil {
		return err
	}
	if response.Code != 0 {
		return fmt.Errorf("飞书应用消息发送失败：%s", firstNotificationError(response.Msg, strconv.Itoa(response.Code)))
	}
	return nil
}

func sendWechatAppApproval(ctx context.Context, value func(string) string, title, message string, tone notificationTone) error {
	var token struct {
		ErrCode int    `json:"errcode"`
		Token   string `json:"access_token"`
	}
	if err := getJSONDecode(ctx, "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid="+url.QueryEscape(value("corpId"))+"&corpsecret="+url.QueryEscape(value("secret")), &token); err != nil {
		return err
	}
	if token.ErrCode != 0 || token.Token == "" {
		return errors.New("企业微信获取访问令牌失败")
	}
	payload := map[string]any{"touser": value("toUser"), "toparty": value("toParty"), "totag": value("toTag"), "agentid": value("agentId"), "msgtype": "markdown", "markdown": map[string]string{"content": wechatNotificationMessage(title, message, tone)}}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := postJSONDecode(ctx, "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+url.QueryEscape(token.Token), payload, nil, &response); err != nil {
		return err
	}
	if response.ErrCode != 0 {
		return fmt.Errorf("企业微信应用消息发送失败：%s", firstNotificationError(response.ErrMsg, strconv.Itoa(response.ErrCode)))
	}
	return nil
}

func sendDingTalkAppApproval(ctx context.Context, value func(string) string, title, message string, tone notificationTone) error {
	var token struct {
		ErrCode int    `json:"errcode"`
		Token   string `json:"access_token"`
	}
	if err := getJSONDecode(ctx, "https://oapi.dingtalk.com/gettoken?appkey="+url.QueryEscape(value("appKey"))+"&appsecret="+url.QueryEscape(value("appSecret")), &token); err != nil {
		return err
	}
	if token.ErrCode != 0 || token.Token == "" {
		return errors.New("钉钉获取访问令牌失败")
	}
	agentID, err := strconv.ParseInt(value("agentId"), 10, 64)
	if err != nil || agentID <= 0 {
		return errors.New("应用 AgentId 必须是数字")
	}
	payload := map[string]any{"agent_id": agentID, "msg": map[string]any{"msgtype": "markdown", "markdown": map[string]string{"title": title, "text": dingTalkApprovalMarkdown(title, message, tone)}}}
	if users := value("useridList"); users != "" {
		payload["userid_list"] = users
	}
	if departments := value("deptIdList"); departments != "" {
		payload["dept_id_list"] = departments
	}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := postJSONDecode(ctx, "https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token="+url.QueryEscape(token.Token), payload, nil, &response); err != nil {
		return err
	}
	if response.ErrCode != 0 {
		return fmt.Errorf("钉钉应用消息发送失败：%s", firstNotificationError(response.ErrMsg, strconv.Itoa(response.ErrCode)))
	}
	return nil
}

func getJSONDecode(ctx context.Context, target string, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return decodeJSONResponse(response, output)
}
func postJSONDecode(ctx context.Context, target string, payload any, headers map[string]string, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return decodeJSONResponse(response, output)
}
func decodeJSONResponse(response *http.Response, output any) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("通知接口返回异常状态：%s", response.Status)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, output)
}

func postJSON(ctx context.Context, method, target string, payload any, headers map[string]string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("通知地址不能为空")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("通知服务响应状态 %d", response.StatusCode)
	}
	return nil
}

func sanitizeChannelConfig(id string, values map[string]any, previous string, box interface {
	Seal(string) (string, error)
	Open(string) (string, error)
}) (map[string]any, string, error) {
	config, secrets := map[string]any{}, map[string]string{}
	if previous != "" {
		secrets, _ = openChannelSecrets(previous, box)
	}
	for key, value := range values {
		if key == "hasSecret" {
			continue
		}
		text, isText := value.(string)
		if isText {
			text = strings.TrimSpace(text)
		}
		if isSecretField(id, key) {
			if text != "" {
				secrets[key] = text
			}
			continue
		}
		if isText && text == "" {
			continue
		}
		config[key] = value
	}
	if id == "webhook" {
		headers, err := normalizeWebhookHeaders(config["headers"])
		if err != nil {
			return nil, "", err
		}
		if len(headers) == 0 {
			delete(config, "headers")
		} else {
			config["headers"] = headers
		}
	}
	if len(secrets) == 0 {
		return config, "", nil
	}
	raw, err := json.Marshal(secrets)
	if err != nil {
		return nil, "", err
	}
	sealed, err := box.Seal(string(raw))
	return config, sealed, err
}

func openChannelSecrets(cipher string, box interface{ Open(string) (string, error) }) (map[string]string, error) {
	values := map[string]string{}
	if cipher == "" {
		return values, nil
	}
	raw, err := box.Open(cipher)
	if err != nil {
		return values, err
	}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return map[string]string{}, nil
	}
	return values, nil
}

func isSecretField(id, key string) bool {
	for _, candidate := range map[string][]string{
		"lark": {"secret"}, "lark_app": {"appSecret"}, "wechat_app": {"secret"}, "dingtalk": {"secret"}, "dingtalk_app": {"appSecret"},
	}[id] {
		if key == candidate {
			return true
		}
	}
	return false
}
