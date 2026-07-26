package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
)

const resetRequestTTL = 15 * time.Minute

var ErrInvalidResetCaptcha = errors.New("验证码不正确")

type passwordResetConfig struct {
	resetCodeTTL    time.Duration
	resetCaptchaTTL time.Duration
	resetCooldown   time.Duration
	rateLimitWindow time.Duration
}

const passwordResetRateLimit = 5

func (s *Service) CreateResetCaptcha(ctx context.Context) (domain.PasswordResetCaptcha, error) {
	left, err := randomNumber(1, 9)
	if err != nil {
		return domain.PasswordResetCaptcha{}, err
	}
	right, err := randomNumber(1, 9)
	if err != nil {
		return domain.PasswordResetCaptcha{}, err
	}
	expiresAt := time.Now().Add(s.passwordResetConfig(ctx).resetCaptchaTTL)
	return domain.PasswordResetCaptcha{
		Token:     s.signResetCaptcha(strconv.FormatInt(left+right, 10), expiresAt),
		Question:  fmt.Sprintf("%d + %d = ?", left, right),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) VerifyResetIdentity(ctx context.Context, username string, captchaToken string, captchaAnswer string) (string, []domain.PasswordResetChannel, error) {
	if !s.verifyResetCaptcha(captchaToken, captchaAnswer) {
		return "", nil, ErrInvalidResetCaptcha
	}
	user, _, err := s.store.FindUser(ctx, strings.TrimSpace(username))
	if err != nil || user.Source != "local" || user.Disabled || strings.TrimSpace(user.Email) == "" {
		return "", nil, errors.New("账号不存在、未配置邮箱或不可用于找回密码")
	}
	email, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil || !email.PasswordResetEnabled {
		return "", nil, errors.New("系统未启用找回密码邮件")
	}
	token, err := randomToken(32)
	if err != nil {
		return "", nil, err
	}
	if err := s.store.CreatePasswordResetRequest(ctx, repository.HashToken(token), user.ID, time.Now().Add(resetRequestTTL)); err != nil {
		return "", nil, err
	}
	channel := domain.PasswordResetChannel{ID: "email", Name: "邮箱验证码", MaskedTo: maskEmail(user.Email), RequiresTo: true}
	return token, []domain.PasswordResetChannel{channel}, nil
}

func (s *Service) signResetCaptcha(answer string, expiresAt time.Time) string {
	payload := answer + ":" + strconv.FormatInt(expiresAt.Unix(), 10)
	mac := hmac.New(sha256.New, s.captchaSecret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + signature))
}

func (s *Service) verifyResetCaptcha(token string, answer string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || !time.Unix(expiresAt, 0).After(time.Now()) {
		return false
	}
	payload := parts[0] + ":" + parts[1]
	mac := hmac.New(sha256.New, s.captchaSecret)
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(parts[2]), []byte(expected)) && hmac.Equal([]byte(strings.TrimSpace(answer)), []byte(parts[0]))
}

func (s *Service) SendResetCode(ctx context.Context, username string, token string, channel string, verifyEmail string, requestIP string) (int, error) {
	request, err := s.store.PasswordResetRequest(ctx, repository.HashToken(strings.TrimSpace(token)))
	if err != nil || request.UsedAt != nil || time.Now().After(request.ExpiresAt) {
		return 0, errors.New("找回密码会话不存在或已过期")
	}
	user, _, err := s.store.FindUser(ctx, strings.TrimSpace(username))
	if err != nil || user.ID != request.UserID || user.Source != "local" || user.Disabled || strings.TrimSpace(user.Email) == "" {
		return 0, errors.New("账号不可用于找回密码")
	}
	if channel != "email" || !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(verifyEmail)) {
		return 0, errors.New("验证邮箱与账号邮箱不一致")
	}
	cfg := s.passwordResetConfig(ctx)
	if request.CodeSentAt != nil {
		remaining := cfg.resetCooldown - time.Since(*request.CodeSentAt)
		if remaining > 0 {
			return int(remaining.Seconds()) + 1, errors.New("验证码发送过于频繁")
		}
	}
	count, err := s.store.CountRecentPasswordResetCodes(ctx, request.UserID, time.Now().Add(-cfg.rateLimitWindow))
	if err != nil {
		return 0, err
	}
	if count >= passwordResetRateLimit {
		return 0, errors.New("验证码请求次数过多，请稍后再试")
	}
	codeNumber, err := randomNumber(100000, 999999)
	if err != nil {
		return 0, err
	}
	code := fmt.Sprintf("%06d", codeNumber)
	hash, err := repository.HashPassword(code)
	if err != nil {
		return 0, err
	}
	expiresAt := time.Now().Add(cfg.resetCodeTTL)
	if err := s.notifier.SendPasswordReset(ctx, user.Email, user.Username, code, expiresAt, requestIP); err != nil {
		return 0, err
	}
	if err := s.store.SetPasswordResetCode(ctx, request.TokenHash, hash, expiresAt); err != nil {
		return 0, err
	}
	return int(cfg.resetCooldown.Seconds()), nil
}

func (s *Service) passwordResetConfig(ctx context.Context) passwordResetConfig {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return defaultPasswordResetConfig()
	}
	return passwordResetConfig{
		resetCodeTTL:    minutesDuration(settings["resetCodeTtlMinutes"], 10, 1, 60),
		resetCaptchaTTL: minutesDuration(settings["resetCaptchaTtlMinutes"], 1, 1, 10),
		resetCooldown:   minutesDuration(settings["passwordResetSendCooldownMinutes"], 0.5, 0.5, 10),
		rateLimitWindow: minutesDuration(settings["passwordResetRateLimitMinutes"], 5, 1, 60),
	}
}

func defaultPasswordResetConfig() passwordResetConfig {
	return passwordResetConfig{
		resetCodeTTL:    10 * time.Minute,
		resetCaptchaTTL: time.Minute,
		resetCooldown:   30 * time.Second,
		rateLimitWindow: 5 * time.Minute,
	}
}

func minutesDuration(value any, fallback float64, minimum float64, maximum float64) time.Duration {
	minutes := fallback
	switch typed := value.(type) {
	case float64:
		minutes = typed
	case int:
		minutes = float64(typed)
	}
	if minutes < minimum || minutes > maximum {
		minutes = fallback
	}
	return time.Duration(minutes * float64(time.Minute))
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, username string, token string, code string, newPassword string) (domain.User, error) {
	if len(newPassword) < 6 || len(newPassword) > 256 {
		return domain.User{}, errors.New("新密码长度必须在 6 到 256 位之间")
	}
	request, err := s.store.PasswordResetRequest(ctx, repository.HashToken(strings.TrimSpace(token)))
	if err != nil || request.UsedAt != nil || time.Now().After(request.ExpiresAt) || request.CodeExpiresAt == nil || time.Now().After(*request.CodeExpiresAt) {
		return domain.User{}, errors.New("找回密码会话或验证码已过期")
	}
	user, _, err := s.store.FindUser(ctx, strings.TrimSpace(username))
	if err != nil || user.ID != request.UserID || user.Source != "local" || user.Disabled || strings.TrimSpace(user.Email) == "" {
		return domain.User{}, errors.New("账号不可用于找回密码")
	}
	if request.CodeHash == "" || repository.VerifyPassword(request.CodeHash, strings.TrimSpace(code)) != nil {
		return domain.User{}, errors.New("邮件验证码不正确")
	}
	hash, err := repository.HashPassword(newPassword)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.store.CompletePasswordReset(ctx, request.TokenHash, request.UserID, hash); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func randomNumber(minimum int64, maximum int64) (int64, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(maximum-minimum+1))
	if err != nil {
		return 0, err
	}
	return value.Int64() + minimum, nil
}

func maskEmail(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "@")
	if len(parts) != 2 || len(parts[0]) < 2 {
		return "***"
	}
	return parts[0][:1] + strings.Repeat("*", max(2, len(parts[0])-1)) + "@" + parts[1]
}
