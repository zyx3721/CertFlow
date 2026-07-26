package notify

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func validateChannelConfig(id string, config map[string]any, secret string) error {
	for _, key := range requiredNotificationFields[id] {
		if isSecretField(id, key) && secret == "" {
			return requiredNotificationFieldError(notificationFieldLabel(id, key))
		}
		if !isSecretField(id, key) && strings.TrimSpace(stringConfig(config, key)) == "" {
			return requiredNotificationFieldError(notificationFieldLabel(id, key))
		}
	}
	if id == "webhook" {
		method := strings.ToUpper(stringConfig(config, "method"))
		if method != "" && method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
			return errors.New("Webhook 请求方法仅支持 POST、PUT 或 PATCH")
		}
	}
	if id == "lark_app" && !isLarkReceiveIDType(stringConfig(config, "receiveIdType")) {
		return errors.New("接收 ID 类型不正确")
	}
	if id == "wechat_app" && stringConfig(config, "toUser") == "" && stringConfig(config, "toParty") == "" && stringConfig(config, "toTag") == "" {
		return errors.New("企业微信接收成员、部门或标签至少填写一项")
	}
	if id == "dingtalk_app" && stringConfig(config, "useridList") == "" && stringConfig(config, "deptIdList") == "" {
		return errors.New("钉钉用户列表或部门列表至少填写一项")
	}
	return nil
}

func isLarkReceiveIDType(value string) bool {
	for _, candidate := range []string{"open_id", "user_id", "union_id", "email", "chat_id"} {
		if value == candidate {
			return true
		}
	}
	return false
}

var requiredNotificationFields = map[string][]string{
	"webhook":      {"url"},
	"lark":         {"webhookUrl"},
	"lark_app":     {"appId", "appSecret", "receiveIdType", "receiveId"},
	"wechat":       {"webhookUrl"},
	"wechat_app":   {"corpId", "agentId", "secret"},
	"dingtalk":     {"webhookUrl", "secret"},
	"dingtalk_app": {"appKey", "appSecret", "agentId"},
}

var notificationFieldLabels = map[string]string{
	"url":           "Webhook URL",
	"webhookUrl":    "机器人 Webhook",
	"appId":         "App ID",
	"appSecret":     "App Secret",
	"receiveIdType": "接收 ID 类型",
	"receiveId":     "接收 ID",
	"corpId":        "企业 ID",
	"agentId":       "AgentId",
	"secret":        "应用 Secret",
	"appKey":        "AppKey",
}

func notificationFieldLabel(id, key string) string {
	if id == "dingtalk" && key == "secret" {
		return "加签密钥"
	}
	if id == "dingtalk_app" && key == "appSecret" {
		return "AppSecret"
	}
	return notificationFieldLabels[key]
}

func requiredNotificationFieldError(label string) error {
	if label == "" {
		label = "配置项"
	}
	last := label[len(label)-1]
	if (last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') {
		return fmt.Errorf("%s 不能为空", label)
	}
	return fmt.Errorf("%s不能为空", label)
}
