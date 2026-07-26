package notify

import (
	"context"
	"fmt"
	"strings"
)

func sendLarkRobot(ctx context.Context, webhookURL string, payload map[string]any) error {
	var response struct {
		Code          int    `json:"code"`
		Msg           string `json:"msg"`
		StatusCode    int    `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}
	if err := postJSONDecode(ctx, webhookURL, payload, nil, &response); err != nil {
		return err
	}
	if response.Code != 0 {
		return fmt.Errorf("飞书机器人消息发送失败：%s", firstNotificationError(response.Msg, fmt.Sprint(response.Code)))
	}
	if response.StatusCode != 0 {
		return fmt.Errorf("飞书机器人消息发送失败：%s", firstNotificationError(response.StatusMessage, fmt.Sprint(response.StatusCode)))
	}
	return nil
}

func sendWechatRobot(ctx context.Context, webhookURL, title, message string, tone notificationTone) error {
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	payload := map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]string{"content": wechatNotificationMessage(title, message, tone)},
	}
	if err := postJSONDecode(ctx, webhookURL, payload, nil, &response); err != nil {
		return err
	}
	if response.ErrCode != 0 {
		return fmt.Errorf("企业微信机器人消息发送失败：%s", firstNotificationError(response.ErrMsg, fmt.Sprint(response.ErrCode)))
	}
	return nil
}

func sendDingTalkRobot(ctx context.Context, webhookURL, title, message string, tone notificationTone) error {
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	payload := map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]string{"title": title, "text": dingTalkApprovalMarkdown(title, message, tone)},
	}
	if err := postJSONDecode(ctx, webhookURL, payload, nil, &response); err != nil {
		return err
	}
	if response.ErrCode != 0 {
		return fmt.Errorf("钉钉机器人消息发送失败：%s", firstNotificationError(response.ErrMsg, fmt.Sprint(response.ErrCode)))
	}
	return nil
}

func dingTalkApprovalMarkdown(title, message string, tone notificationTone) string {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " ") + "  "
	}
	color := "green"
	if isWarningTone(tone) {
		color = "orange"
	}
	return "## " + strings.TrimSpace(title) + "  \n<font color=\"" + color + "\">" + strings.Join(lines, "\n") + "</font>  "
}
