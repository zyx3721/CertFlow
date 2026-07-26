package notify

import (
	"html"
	"strings"
)

type notificationTone string

const (
	notificationToneNormal  notificationTone = "normal"
	notificationToneWarning notificationTone = "warning"
)

func isWarningTone(tone notificationTone) bool {
	return tone == notificationToneWarning
}

func larkNotificationColor(tone notificationTone) string {
	if isWarningTone(tone) {
		return "orange"
	}
	return "green"
}

func larkHeaderTemplate(tone notificationTone) string {
	if isWarningTone(tone) {
		return "red"
	}
	return "green"
}

func larkNotificationMessage(message string, tone notificationTone) string {
	return "<font color='" + larkNotificationColor(tone) + "'>" + strings.TrimSpace(message) + "</font>"
}

func wechatNotificationMessage(title, message string, tone notificationTone) string {
	color := "info"
	if isWarningTone(tone) {
		color = "warning"
	}
	return "## " + strings.TrimSpace(title) + "\n<font color=\"" + color + "\">" + strings.TrimSpace(message) + "</font>"
}

func notificationEmailBody(message string, tone notificationTone) string {
	color := "#2BDE3F"
	if isWarningTone(tone) {
		color = "#FFC007"
	}
	return `<p style="margin:0;color:` + color + `;font-size:15px;line-height:1.8;">` + html.EscapeString(message) + `</p>`
}
