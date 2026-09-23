package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"certflow/backend/internal/repository"
)

const (
	WecomModeDirect = "direct"
	WecomModeSSO    = "sso"

	// defaultWecomStateTTL 是企微授权 state 的默认有效期，可在基础配置「安全时效」中调整。
	defaultWecomStateTTL = 5 * time.Minute
)

// WecomEmbedCallbackPath 内嵌二维码登录的回跳路由路径，与整页登录回跳的 /login 区分
const WecomEmbedCallbackPath = "/wecom-qr-callback"

var (
	ErrWecomNotEnabled = errors.New("企业微信认证未启用")
	ErrWecomNotBound   = errors.New("该企业微信账号尚未绑定系统用户")
	ErrWecomConflict   = errors.New("该企业微信账号已绑定其他用户")
	ErrWecomState      = errors.New("企业微信登录状态已过期，请重新扫码")
)

// WecomConfig 是解密后的运行时企微认证配置。
type WecomConfig struct {
	Enabled        bool
	Mode           string
	CorpID         string
	AgentID        string
	Secret         string
	RedirectPrefix string
	SSOBaseURL     string
	SSOAppID       string
	SSOAppSecret   string
}

type wecomState struct {
	Purpose string `json:"p"`
	UserID  string `json:"u,omitempty"`
	Exp     int64  `json:"e"`
	Nonce   string `json:"n"`
}

func (s *Service) runtimeWecomConfig(ctx context.Context) (WecomConfig, error) {
	setting, err := s.store.AuthProviderSetting(ctx, "wecom")
	if errors.Is(err, repository.ErrNotFound) {
		return WecomConfig{}, nil
	}
	if err != nil {
		return WecomConfig{}, err
	}
	cfg, err := runtimeWecomConfigFromSetting(setting, s.box.Open)
	if err != nil {
		return cfg, err
	}
	if cfg.Enabled {
		if err := cfg.validate(); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func (c WecomConfig) validate() error {
	switch c.Mode {
	case WecomModeDirect:
		if c.CorpID == "" || c.AgentID == "" || c.Secret == "" {
			return errors.New("直连模式必须配置企业 ID、应用 AgentId 和应用 Secret")
		}
	case WecomModeSSO:
		if !strings.HasPrefix(c.SSOBaseURL, "http://") && !strings.HasPrefix(c.SSOBaseURL, "https://") {
			return errors.New("统一认证中心地址必须以 http:// 或 https:// 开头")
		}
		if c.SSOAppID == "" || c.SSOAppSecret == "" {
			return errors.New("统一认证中心模式必须配置应用标识和应用 Secret")
		}
	default:
		return errors.New("认证方式仅支持直连或统一认证中心")
	}
	return nil
}

// wecomStateTTL 读取基础配置中的企业微信扫码有效期，缺省或越界时回落默认值。
func (s *Service) wecomStateTTL(ctx context.Context) time.Duration {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return defaultWecomStateTTL
	}
	minutes, ok := settings["wecomStateTtlMinutes"].(float64)
	if !ok || minutes < 1 || minutes > 60 {
		return defaultWecomStateTTL
	}
	return time.Duration(minutes) * time.Minute
}

// stateSecret 复用会话签名密钥为企微授权 state 做 HMAC 签名。
func (s *Service) stateSecret() string {
	return string(s.captchaSecret)
}

func wecomHMAC(secret, message string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func signWecomState(secret string, state wecomState, ttl time.Duration) (string, error) {
	state.Exp = time.Now().Add(ttl).Unix()
	return signWecomStatePayload(secret, state)
}

func signWecomStatePayload(secret string, state wecomState) (string, error) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	state.Nonce = hex.EncodeToString(nonce)
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	body := hex.EncodeToString(payload)
	return body + "." + wecomHMAC(secret, body), nil
}

func verifyWecomState(secret, state, purpose string) (wecomState, error) {
	body, signature, found := strings.Cut(state, ".")
	if !found || !hmac.Equal([]byte(wecomHMAC(secret, body)), []byte(signature)) {
		return wecomState{}, ErrWecomState
	}
	payload, err := hex.DecodeString(body)
	if err != nil {
		return wecomState{}, ErrWecomState
	}
	var parsed wecomState
	if err := json.Unmarshal(payload, &parsed); err != nil || parsed.Purpose != purpose {
		return wecomState{}, ErrWecomState
	}
	if time.Now().Unix() > parsed.Exp {
		return wecomState{}, ErrWecomState
	}
	return parsed, nil
}

// wecomCallbackRedirectURL 回调落地页固定为前端登录路由：配置了前缀用前缀，否则按当前访问地址推断。
func (c WecomConfig) wecomCallbackRedirectURL(scheme, host string) string {
	return c.wecomCallbackRedirectURLPath(scheme, host, "/login")
}

// wecomCallbackRedirectURLPath 回调落地页按指定路径构造：配置了前缀用前缀，否则按当前访问地址推断。
func (c WecomConfig) wecomCallbackRedirectURLPath(scheme, host, path string) string {
	if c.RedirectPrefix != "" {
		return strings.TrimRight(c.RedirectPrefix, "/") + path
	}
	return scheme + "://" + host + path
}

// ssoLoginURL 构造统一认证中心登录入口地址。
func (c WecomConfig) ssoLoginURL() string {
	return strings.TrimSuffix(c.SSOBaseURL, "/") + "/login?app=" + queryEscape(c.SSOAppID)
}

// wecomAuthorizePayload 按认证方式构造登录跳转地址与内嵌二维码参数：SSO 复用认证中心入口地址，直连区分整页与内嵌回跳并携带已签发 state。
func (c WecomConfig) wecomAuthorizePayload(scheme, host, state string) WecomAuthorizePayload {
	if c.Mode == WecomModeSSO {
		target := c.ssoLoginURL()
		return WecomAuthorizePayload{AuthMode: WecomModeSSO, URL: target, EmbedURL: target, CallbackPath: WecomEmbedCallbackPath}
	}
	return WecomAuthorizePayload{
		AuthMode:     WecomModeDirect,
		URL:          c.authorizeURL(c.wecomCallbackRedirectURLPath(scheme, host, "/login"), state),
		EmbedURL:     c.authorizeURL(c.wecomCallbackRedirectURLPath(scheme, host, WecomEmbedCallbackPath), state),
		State:        state,
		CallbackPath: WecomEmbedCallbackPath,
	}
}

// authorizeURL 构造企微扫码授权页地址。
func (c WecomConfig) authorizeURL(redirectURI, state string) string {
	values := fmt.Sprintf("login_type=CorpApp&appid=%s&agentid=%s&redirect_uri=%s&state=%s",
		c.CorpID, c.AgentID, queryEscape(redirectURI), queryEscape(state))
	return "https://login.work.weixin.qq.com/wwlogin/sso/login?" + values
}
