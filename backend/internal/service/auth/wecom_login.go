package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
)

// WecomAuthorizePayload 企业微信扫码登录跳转地址与内嵌二维码渲染参数，EmbedURL 为空表示仅支持整页跳转。
type WecomAuthorizePayload struct {
	AuthMode     string
	URL          string
	EmbedURL     string
	State        string
	CallbackPath string
}

// NewWecomAuthorizePayload 构造企业微信扫码登录跳转与内嵌二维码参数：SSO 复用认证中心入口地址，直连签发 state 并区分整页与内嵌回跳。
func (s *Service) NewWecomAuthorizePayload(ctx context.Context, scheme, host string) (WecomAuthorizePayload, error) {
	cfg, err := s.runtimeWecomConfig(ctx)
	if err != nil {
		return WecomAuthorizePayload{}, err
	}
	if !cfg.Enabled {
		return WecomAuthorizePayload{}, ErrWecomNotEnabled
	}
	if cfg.Mode == WecomModeSSO {
		return cfg.wecomAuthorizePayload(scheme, host, ""), nil
	}
	state, err := s.newWecomState(ctx, "login", "")
	if err != nil {
		return WecomAuthorizePayload{}, err
	}
	return cfg.wecomAuthorizePayload(scheme, host, state), nil
}

// WecomBindURL 为当前登录用户签发绑定用授权地址。
func (s *Service) WecomBindURL(ctx context.Context, user domain.User, scheme, host string) (string, error) {
	cfg, err := s.runtimeWecomConfig(ctx)
	if err != nil {
		return "", err
	}
	if !cfg.Enabled {
		return "", ErrWecomNotEnabled
	}
	if cfg.Mode == WecomModeSSO {
		return cfg.ssoLoginURL(), nil
	}
	state, err := s.newWecomState(ctx, "bind", user.ID)
	if err != nil {
		return "", err
	}
	redirectURI := cfg.wecomCallbackRedirectURL(scheme, host)
	return cfg.authorizeURL(redirectURI, state), nil
}

func (s *Service) newWecomState(ctx context.Context, purpose, userID string) (string, error) {
	return signWecomState(s.stateSecret(), wecomState{Purpose: purpose, UserID: userID}, s.wecomStateTTL(ctx))
}

// WecomResolveCode 校验直连模式回传的 code/state 并换取企微 userid。
// 返回的第一个值是 state 中记录的绑定发起用户 ID（仅 bind 用途非空）。
func (s *Service) WecomResolveCode(ctx context.Context, code, state, purpose string) (string, string, error) {
	cfg, err := s.runtimeWecomConfig(ctx)
	if err != nil {
		return "", "", err
	}
	if !cfg.Enabled || cfg.Mode != WecomModeDirect {
		return "", "", ErrWecomNotEnabled
	}
	signed, err := verifyWecomState(s.stateSecret(), state, purpose)
	if err != nil {
		return "", "", err
	}
	userid, err := wecomExchangeCode(ctx, cfg.CorpID, cfg.Secret, code)
	if err != nil {
		return "", "", err
	}
	return signed.UserID, userid, nil
}

// WecomResolveTicket 校验统一认证中心回传的 ticket 并换取企微 userid。
func (s *Service) WecomResolveTicket(ctx context.Context, ticket string) (string, error) {
	cfg, err := s.runtimeWecomConfig(ctx)
	if err != nil {
		return "", err
	}
	if !cfg.Enabled || cfg.Mode != WecomModeSSO {
		return "", ErrWecomNotEnabled
	}
	return wecomVerifyTicket(ctx, cfg.SSOBaseURL, cfg.SSOAppID, cfg.SSOAppSecret, ticket)
}

// LoginByWecom 校验绑定关系并签发平台会话。
func (s *Service) LoginByWecom(ctx context.Context, wecomUserid string) (domain.Session, error) {
	user, err := s.store.FindUserByWecomBinding(ctx, wecomUserid)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Session{}, ErrWecomNotBound
	}
	if err != nil {
		return domain.Session{}, err
	}
	if user.Disabled {
		return domain.Session{}, ErrUserNotProvisioned
	}
	user.AuthenticationProvider = "wecom"
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return domain.Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(s.ttl)
	if err = s.store.CreateSession(ctx, token, user.ID, "wecom", expires); err != nil {
		return domain.Session{}, err
	}
	_ = s.store.DeleteExpiredSessions(ctx)
	if err = s.store.RecordUserLogin(ctx, user.ID); err != nil {
		return domain.Session{}, err
	}
	return domain.Session{Token: token, User: user, ExpiresAt: expires}, nil
}

// WecomBindingOf 返回当前用户绑定的企微 userid，未绑定时返回空串。
func (s *Service) WecomBindingOf(ctx context.Context, user domain.User) (string, bool, error) {
	userid, err := s.store.WecomBindingOfUser(ctx, user.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userid, true, nil
}

// BindWecom 将企微 userid 绑定到当前用户；一个企微账号只能绑定一个用户，换绑覆盖自身旧绑定。
func (s *Service) BindWecom(ctx context.Context, user domain.User, wecomUserid string) (string, error) {
	owner, err := s.store.FindWecomBindingOwner(ctx, wecomUserid)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return "", err
	}
	if err == nil && owner != user.ID {
		return "", ErrWecomConflict
	}
	if err := s.store.SaveWecomBinding(ctx, user.ID, wecomUserid); err != nil {
		return "", err
	}
	return wecomUserid, nil
}

// UnbindWecom 解除当前用户的企微绑定，未绑定时返回 false。
func (s *Service) UnbindWecom(ctx context.Context, user domain.User) (bool, error) {
	return s.store.DeleteWecomBinding(ctx, user.ID)
}
