package router

import (
	"net/http"
	"strconv"
	"strings"
)

func (r *Router) passwordResetCaptcha(w http.ResponseWriter, q *http.Request) {
	captcha, err := r.auth.CreateResetCaptcha(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "生成图形验证码失败"})
		return
	}
	write(w, http.StatusOK, captcha)
}

func (r *Router) passwordResetVerify(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Username      string `json:"username"`
		CaptchaToken  string `json:"captchaToken"`
		CaptchaAnswer string `json:"captchaAnswer"`
	}
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.Username) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	token, channels, err := r.auth.VerifyResetIdentity(q.Context(), body.Username, body.CaptchaToken, body.CaptchaAnswer)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, map[string]any{"verificationToken": token, "channels": channels})
}

func (r *Router) passwordResetSend(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Username          string `json:"username"`
		VerificationToken string `json:"verificationToken"`
		Channel           string `json:"channel"`
		VerifyEmail       string `json:"verifyEmail"`
	}
	if err := decode(q, &body); err != nil || body.Channel != "email" || strings.TrimSpace(body.VerificationToken) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	if strings.TrimSpace(body.Username) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "用户名不能为空"})
		return
	}
	if strings.TrimSpace(body.VerifyEmail) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请输入验证邮箱"})
		return
	}
	cooldown, err := r.auth.SendResetCode(q.Context(), body.Username, body.VerificationToken, body.Channel, body.VerifyEmail, clientIP(q))
	if err != nil {
		if cooldown > 0 {
			write(w, http.StatusTooManyRequests, map[string]any{"message": "验证码已发送，请于 " + strconv.Itoa(cooldown) + " 秒后再试", "cooldownSeconds": cooldown})
			return
		}
		if strings.Contains(err.Error(), "请求次数过多") {
			write(w, http.StatusTooManyRequests, map[string]any{"message": "验证码请求过于频繁，请稍后再试", "cooldownSeconds": cooldown})
			return
		}
		write(w, http.StatusBadRequest, map[string]any{"message": err.Error(), "cooldownSeconds": cooldown})
		return
	}
	write(w, http.StatusOK, map[string]any{"status": "ok", "cooldownSeconds": cooldown})
}

func (r *Router) passwordResetConfirm(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Username          string `json:"username"`
		VerificationToken string `json:"verificationToken"`
		Code              string `json:"code"`
		NewPassword       string `json:"newPassword"`
		ConfirmPassword   string `json:"confirmPassword"`
	}
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.VerificationToken) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	if strings.TrimSpace(body.Username) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "用户名不能为空"})
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请输入找回密码验证码"})
		return
	}
	if len(body.NewPassword) < 6 || body.NewPassword != body.ConfirmPassword {
		write(w, http.StatusBadRequest, map[string]string{"message": "新密码至少 6 位且两次输入必须一致"})
		return
	}
	user, err := r.auth.ConfirmPasswordReset(q.Context(), body.Username, body.VerificationToken, body.Code, body.NewPassword)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	cleared, clearErr := r.auth.ClearLoginFailures(q.Context(), body.Username)
	if clearErr != nil {
		r.logger.Error("Clear login failures failed", "error", clearErr)
	}
	detail := "用户 " + user.Username + " 已通过找回密码流程修改密码"
	if cleared > 0 {
		detail += "并解除登录失败锁定"
	}
	r.store.Audit(q.Context(), user, "找回密码成功", user.Username, "auth", "success", clientIP(q), detail)
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}
