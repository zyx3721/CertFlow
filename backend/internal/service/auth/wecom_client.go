package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	wecomAPIBase      = "https://qyapi.weixin.qq.com"
	wecomTokenTTLOff  = 7000 * time.Second
	wecomHTTPTimeout  = 10 * time.Second
	wecomBodyLimit    = 1 << 20
	wecomTokenRetries = 1
)

var wecomHTTPClient = &http.Client{Timeout: wecomHTTPTimeout}

type wecomCachedToken struct {
	token     string
	expiresAt time.Time
}

var wecomTokenCache = struct {
	sync.Mutex
	items map[string]wecomCachedToken
}{items: map[string]wecomCachedToken{}}

func queryEscape(value string) string { return url.QueryEscape(value) }

type wecomAPIError struct {
	Code int    `json:"errcode"`
	Msg  string `json:"errmsg"`
}

func (e wecomAPIError) Error() string { return fmt.Sprintf("wecom errcode %d: %s", e.Code, e.Msg) }

func wecomUserMessage(err error) string {
	var apiErr wecomAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 40029:
			return "授权码无效或已被使用，请重新扫码"
		case 60020:
			return "企业微信应用未放行服务器出口 IP，请在企微后台配置企业可信 IP"
		case 40013:
			return "企业 ID 不正确，请检查直连配置"
		case 40001, 41001:
			return "应用 Secret 不正确，请检查直连配置"
		}
		return "企业微信接口返回错误：" + apiErr.Msg
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "企业微信服务连接超时，请稍后重试"
	}
	return "企业微信服务暂时不可用，请稍后重试"
}

func wecomCachedTokenKey(corpID, secret string) string {
	sum := sha256.Sum256([]byte(corpID + "\x00" + secret))
	return hex.EncodeToString(sum[:])
}

func wecomAccessToken(ctx context.Context, corpID, secret string) (string, error) {
	key := wecomCachedTokenKey(corpID, secret)
	wecomTokenCache.Lock()
	cached, ok := wecomTokenCache.items[key]
	wecomTokenCache.Unlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.token, nil
	}
	token, err := fetchWecomToken(ctx, corpID, secret)
	if err != nil {
		return "", err
	}
	wecomTokenCache.Lock()
	wecomTokenCache.items[key] = wecomCachedToken{token: token, expiresAt: time.Now().Add(wecomTokenTTLOff)}
	wecomTokenCache.Unlock()
	return token, nil
}

func fetchWecomToken(ctx context.Context, corpID, secret string) (string, error) {
	endpoint := wecomAPIBase + "/cgi-bin/gettoken?corpid=" + queryEscape(corpID) + "&corpsecret=" + queryEscape(secret)
	body, err := wecomGetJSON(ctx, endpoint)
	if err != nil {
		return "", err
	}
	var result struct {
		wecomAPIError
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", result.wecomAPIError
	}
	if result.AccessToken == "" {
		return "", errors.New("wecom access token is empty")
	}
	return result.AccessToken, nil
}

// wecomExchangeCode 用企微 OAuth code 换取企业成员 userid。
func wecomExchangeCode(ctx context.Context, corpID, secret, code string) (string, error) {
	token, err := wecomAccessToken(ctx, corpID, secret)
	if err != nil {
		return "", err
	}
	userid, err := fetchWecomUserid(ctx, token, code)
	if err == nil {
		return userid, nil
	}
	var apiErr wecomAPIError
	if !errors.As(err, &apiErr) || (apiErr.Code != 40014 && apiErr.Code != 42001) {
		return "", err
	}
	wecomTokenCache.Lock()
	delete(wecomTokenCache.items, wecomCachedTokenKey(corpID, secret))
	wecomTokenCache.Unlock()
	token, err = wecomAccessToken(ctx, corpID, secret)
	if err != nil {
		return "", err
	}
	return fetchWecomUserid(ctx, token, code)
}

func fetchWecomUserid(ctx context.Context, token, code string) (string, error) {
	endpoint := wecomAPIBase + "/cgi-bin/auth/getuserinfo?access_token=" + queryEscape(token) + "&code=" + queryEscape(code)
	body, err := wecomGetJSON(ctx, endpoint)
	if err != nil {
		return "", err
	}
	var result struct {
		wecomAPIError
		UserID string `json:"userid"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", result.wecomAPIError
	}
	if result.UserID == "" {
		return "", errors.New("wecom userid is empty")
	}
	return result.UserID, nil
}

func wecomGetJSON(ctx context.Context, endpoint string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := wecomHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(io.LimitReader(response.Body, wecomBodyLimit))
}

type wecomSSOVerifyResponse struct {
	Userid string `json:"userid"`
	Name   string `json:"name"`
	Error  string `json:"error"`
}

// wecomVerifyTicket 向统一认证中心校验 ticket 并换取企微 userid。
// 签名格式与认证中心约定一致：hex(HMAC-SHA256(app_secret, app + "\n" + ticket + "\n" + ts))。
func wecomVerifyTicket(ctx context.Context, baseURL, appID, appSecret, ticket string) (string, error) {
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(appSecret))
	fmt.Fprintf(mac, "%s\n%s\n%d", appID, ticket, ts)
	payload := map[string]any{"app": appID, "ticket": ticket, "ts": ts, "sign": hex.EncodeToString(mac.Sum(nil))}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/api/verify"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := wecomHTTPClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, wecomBodyLimit))
	if err != nil {
		return "", err
	}
	var result wecomSSOVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", wecomSSOError(result.Error, response.StatusCode)
	}
	if result.Userid == "" {
		return "", errors.New("auth center returned empty userid")
	}
	return result.Userid, nil
}

func wecomSSOError(code string, status int) error {
	switch code {
	case "invalid_app":
		return errors.New("统一认证中心未登记本系统应用标识")
	case "invalid_sign":
		return errors.New("统一认证中心应用 Secret 校验失败，请核对配置")
	case "expired_ts":
		return errors.New("系统时钟与统一认证中心偏差过大，请校准服务器时间后重试")
	case "invalid_ticket":
		return errors.New("登录凭证无效或已过期，请重新扫码")
	}
	return fmt.Errorf("统一认证中心校验失败（HTTP %d）", status)
}
