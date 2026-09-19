package router

import (
	"errors"
	"net/http"
	"strings"

	"certflow/backend/internal/domain"
	authsvc "certflow/backend/internal/service/auth"
)

type wecomAuthorizeResponse struct {
	URL string `json:"url"`
}

type wecomCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type wecomSSOCallbackRequest struct {
	Ticket string `json:"ticket"`
}

type wecomBindingResponse struct {
	Bound       bool   `json:"bound"`
	WecomUserid string `json:"wecomUserid,omitempty"`
}

// requestScheme 推断请求对外协议，反代场景依赖 X-Forwarded-Proto。
func requestScheme(request *http.Request) string {
	if proto := strings.TrimSpace(request.Header.Get("X-Forwarded-Proto")); proto != "" {
		return proto
	}
	if request.TLS != nil {
		return "https"
	}
	return "http"
}

// requestHost 返回对外主机名，优先使用反代透传的 Host。
func requestHost(request *http.Request) string {
	if host := strings.TrimSpace(request.Header.Get("X-Forwarded-Host")); host != "" {
		return host
	}
	return request.Host
}

// swaggerWecomAuthorize godoc
// @Summary 获取企业微信扫码授权地址
// @Description 根据企业微信认证配置返回扫码登录地址，直连模式返回企微 wwlogin 页面，统一认证中心模式返回认证中心登录页。
// @Tags Auth
// @Produce json
// @Success 200 {object} wecomAuthorizeResponse
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/auth/wecom/authorize [get]
func (r *Router) wecomAuthorize(w http.ResponseWriter, q *http.Request) {
	url, err := r.auth.WecomAuthorizeURL(q.Context(), requestScheme(q), requestHost(q))
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, wecomAuthorizeResponse{URL: url})
}

// swaggerWecomCallback godoc
// @Summary 企业微信扫码登录回调
// @Description 直连模式：校验 state 并用授权码换取企微 userid，仅允许已绑定平台用户的企业微信账号登录。
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body wecomCallbackRequest true "企微回跳参数"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/auth/wecom/callback [post]
func (r *Router) wecomCallback(w http.ResponseWriter, q *http.Request) {
	var body wecomCallbackRequest
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.State) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	_, userid, err := r.auth.WecomResolveCode(q.Context(), strings.TrimSpace(body.Code), strings.TrimSpace(body.State), "login")
	if err != nil {
		r.recordWecomLoginFailure(q, err)
		write(w, http.StatusBadRequest, map[string]string{"message": wecomLoginFailureMessage(err)})
		return
	}
	r.completeWecomLogin(w, q, userid)
}

// swaggerWecomSSOCallback godoc
// @Summary 统一认证中心登录回调
// @Description 统一认证中心模式：使用 ticket 调用认证中心换取企微 userid，仅允许已绑定平台用户的企业微信账号登录。
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body wecomSSOCallbackRequest true "认证中心回跳 ticket"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/auth/wecom/sso/callback [post]
func (r *Router) wecomSSOCallback(w http.ResponseWriter, q *http.Request) {
	var body wecomSSOCallbackRequest
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.Ticket) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	userid, err := r.auth.WecomResolveTicket(q.Context(), strings.TrimSpace(body.Ticket))
	if err != nil {
		r.recordWecomLoginFailure(q, err)
		write(w, http.StatusBadRequest, map[string]string{"message": wecomLoginFailureMessage(err)})
		return
	}
	r.completeWecomLogin(w, q, userid)
}

func (r *Router) completeWecomLogin(w http.ResponseWriter, q *http.Request, userid string) {
	session, err := r.auth.LoginByWecom(q.Context(), userid)
	if err != nil {
		r.store.Audit(q.Context(), domain.User{Username: "-"}, "用户登录", userid, "auth", "failure", clientIP(q), "企业微信账号 "+userid+" 使用企业微信方式登录失败："+wecomLoginFailureMessage(err))
		write(w, http.StatusUnauthorized, map[string]string{"message": wecomLoginFailureMessage(err)})
		return
	}
	r.store.Audit(q.Context(), session.User, "用户登录", session.User.Username, "auth", "success", clientIP(q), authenticationAuditDetail(session.User, "登录"))
	write(w, http.StatusOK, session)
}

func (r *Router) recordWecomLoginFailure(q *http.Request, err error) {
	r.store.Audit(q.Context(), domain.User{Username: "-"}, "用户登录", "-", "auth", "failure", clientIP(q), "使用企业微信方式登录失败："+err.Error())
}

func wecomLoginFailureMessage(err error) string {
	switch {
	case errors.Is(err, authsvc.ErrWecomState):
		return "企业微信登录状态已过期，请重新扫码"
	case errors.Is(err, authsvc.ErrWecomNotBound):
		return "该企业微信账号尚未绑定系统用户"
	case errors.Is(err, authsvc.ErrUserNotProvisioned):
		return "用户未在平台中启用"
	default:
		return err.Error()
	}
}

// swaggerWecomBindURL godoc
// @Summary 获取企业微信绑定授权地址
// @Description 为当前登录用户签发绑定用扫码地址，state 与当前用户绑定。
// @Tags Auth
// @Produce json
// @Success 200 {object} wecomAuthorizeResponse
// @Failure 400 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/auth/wecom/bind-url [get]
func (r *Router) wecomBindURL(w http.ResponseWriter, q *http.Request) {
	url, err := r.auth.WecomBindURL(q.Context(), current(q), requestScheme(q), requestHost(q))
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusOK, wecomAuthorizeResponse{URL: url})
}

// swaggerWecomBind godoc
// @Summary 绑定企业微信
// @Description 直连模式：校验绑定 state 并用授权码换取企微 userid 后绑定到当前用户。
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body wecomCallbackRequest true "企微回跳参数"
// @Success 200 {object} wecomBindingResponse
// @Failure 400 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/auth/wecom/bind [post]
func (r *Router) wecomBind(w http.ResponseWriter, q *http.Request) {
	var body wecomCallbackRequest
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.State) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	stateUser, userid, err := r.auth.WecomResolveCode(q.Context(), strings.TrimSpace(body.Code), strings.TrimSpace(body.State), "bind")
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": wecomLoginFailureMessage(err)})
		return
	}
	if stateUser != "" && stateUser != current(q).ID {
		write(w, http.StatusForbidden, map[string]string{"message": "绑定状态与当前用户不匹配，请重新发起绑定"})
		return
	}
	r.completeWecomBind(w, q, userid)
}

// swaggerWecomSSOBind godoc
// @Summary 通过统一认证中心绑定企业微信
// @Description 统一认证中心模式：使用 ticket 换取企微 userid 后绑定到当前用户。
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body wecomSSOCallbackRequest true "认证中心回跳 ticket"
// @Success 200 {object} wecomBindingResponse
// @Failure 400 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/auth/wecom/sso/bind [post]
func (r *Router) wecomSSOBind(w http.ResponseWriter, q *http.Request) {
	var body wecomSSOCallbackRequest
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.Ticket) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	userid, err := r.auth.WecomResolveTicket(q.Context(), strings.TrimSpace(body.Ticket))
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": wecomLoginFailureMessage(err)})
		return
	}
	r.completeWecomBind(w, q, userid)
}

func (r *Router) completeWecomBind(w http.ResponseWriter, q *http.Request, userid string) {
	user := current(q)
	bound, err := r.auth.BindWecom(q.Context(), user, userid)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, authsvc.ErrWecomConflict) {
			status = http.StatusConflict
		}
		write(w, status, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), user, "绑定企业微信", user.Username, "auth", "success", clientIP(q), "系统用户 "+user.Username+" 绑定企业微信账号 "+bound+" 成功")
	write(w, http.StatusOK, wecomBindingResponse{Bound: true, WecomUserid: bound})
}

// swaggerWecomUnbind godoc
// @Summary 解绑企业微信
// @Description 解除当前用户的企业微信账号绑定。
// @Tags Auth
// @Produce json
// @Success 200 {object} wecomBindingResponse
// @Failure 500 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/auth/wecom/bind [delete]
func (r *Router) wecomUnbind(w http.ResponseWriter, q *http.Request) {
	user := current(q)
	previousUserid, _, _ := r.auth.WecomBindingOf(q.Context(), user)
	removed, err := r.auth.UnbindWecom(q.Context(), user)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "解绑企业微信失败"})
		return
	}
	if removed {
		r.store.Audit(q.Context(), user, "解绑企业微信", user.Username, "auth", "success", clientIP(q), "系统用户 "+user.Username+" 已解除绑定企业微信账号 "+previousUserid)
	}
	write(w, http.StatusOK, wecomBindingResponse{Bound: false})
}

// swaggerGetWecomProvider godoc
// @Summary 获取企业微信认证配置
// @Description 读取企业微信认证配置，Secret 只返回是否已配置（hasSecret / hasSsoAppSecret）。
// @Tags Settings
// @Produce json
// @Success 200 {object} wecomProviderResponse
// @Failure 500 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/settings/auth-provider/wecom [get]
func (r *Router) getWecomProvider(w http.ResponseWriter, q *http.Request) {
	item, err := r.auth.WecomSetting(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取企业微信认证配置失败"})
		return
	}
	write(w, http.StatusOK, item)
}

// swaggerSaveWecomProvider godoc
// @Summary 保存企业微信认证配置
// @Description 保存企业微信认证配置；Secret 留空表示沿用旧值，clearConfig 为 true 时清空并停用。
// @Tags Settings
// @Accept json
// @Produce json
// @Param body body configurationRequest true "企业微信认证配置"
// @Success 200 {object} wecomProviderResponse
// @Failure 400 {object} errorDocResponse
// @Security BearerAuth
// @Router /api/v1/settings/auth-provider/wecom [put]
func (r *Router) saveWecomProvider(w http.ResponseWriter, q *http.Request) {
	var body configurationRequest
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	var item authsvc.WecomSettingView
	var err error
	if body.ClearConfig {
		item, err = r.auth.ClearWecomSetting(q.Context())
	} else {
		item, err = r.auth.SaveWecomSetting(q.Context(), body.Name, body.Enabled, body.Config)
	}
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), current(q), "更新认证配置", item.Name, "settings", "success", clientIP(q), item.Name+" 认证配置已更新")
	write(w, http.StatusOK, item)
}
