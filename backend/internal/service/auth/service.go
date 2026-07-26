package auth

import (
	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
	"certflow/backend/internal/security"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

var (
	ErrCredentials        = errors.New("用户名或密码错误")
	ErrUserNotProvisioned = errors.New("external user is not provisioned")
)

type Service struct {
	store         *repository.Store
	ttl           time.Duration
	idleTTL       time.Duration
	box           *security.Cryptobox
	captchaSecret []byte
	notifier      PasswordResetNotifier
}

type PasswordResetNotifier interface {
	SendPasswordReset(context.Context, string, string, string, time.Time, string) error
}

func New(store *repository.Store, ttl time.Duration, idleTTL time.Duration, sessionSecret string, box *security.Cryptobox, notifier PasswordResetNotifier) *Service {
	return &Service{store: store, ttl: ttl, idleTTL: idleTTL, captchaSecret: []byte(sessionSecret), box: box, notifier: notifier}
}
func (s *Service) Login(ctx context.Context, username, password, provider string) (domain.Session, error) {
	authProvider := "local"
	user, err := s.loginLocal(ctx, username, password)
	if provider == "ldap" {
		authProvider = "ldap"
		user, err = s.loginLDAP(ctx, username, password)
	}
	if err != nil {
		return domain.Session{}, err
	}
	if user.Disabled {
		return domain.Session{}, ErrCredentials
	}
	user.AuthenticationProvider = authProvider
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return domain.Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(s.ttl)
	if err = s.store.CreateSession(ctx, token, user.ID, authProvider, expires); err != nil {
		return domain.Session{}, err
	}
	if err = s.store.RecordUserLogin(ctx, user.ID); err != nil {
		return domain.Session{}, err
	}
	return domain.Session{Token: token, User: user, ExpiresAt: expires}, nil
}
func (s *Service) loginLocal(ctx context.Context, username, password string) (domain.User, error) {
	user, hash, err := s.store.FindUser(ctx, strings.TrimSpace(username))
	if err != nil || repository.VerifyPassword(hash, password) != nil {
		return domain.User{}, ErrCredentials
	}
	return user, nil
}
func (s *Service) loginLDAP(ctx context.Context, username, password string) (domain.User, error) {
	cfg, err := s.runtimeLDAPConfig(ctx)
	if err != nil {
		return domain.User{}, ErrCredentials
	}
	if !cfg.Enabled || cfg.Host == "" || cfg.BaseDN == "" || password == "" {
		return domain.User{}, ErrCredentials
	}
	conn, err := ldapConn(ctx, cfg)
	if err != nil {
		return domain.User{}, ErrCredentials
	}
	defer conn.Close()
	if cfg.BindDN != "" && conn.Bind(cfg.BindDN, cfg.BindPassword) != nil {
		return domain.User{}, ErrCredentials
	}
	filter := ldapLoginUserFilter(cfg, strings.TrimSpace(username))
	result, err := conn.Search(ldap.NewSearchRequest(cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, cfg.TimeoutSeconds, false, filter, []string{cfg.UsernameAttribute, cfg.DisplayNameAttribute, cfg.EmailAttribute}, nil))
	if err != nil || len(result.Entries) != 1 {
		return domain.User{}, ErrCredentials
	}
	entry := result.Entries[0]
	if conn.Bind(entry.DN, password) != nil {
		return domain.User{}, ErrCredentials
	}
	resolved := entry.GetAttributeValue(cfg.UsernameAttribute)
	if resolved == "" {
		resolved = username
	}
	user, _, err := s.store.FindUser(ctx, resolved)
	if err != nil || user.Disabled {
		return domain.User{}, ErrUserNotProvisioned
	}
	return user, nil
}
func ldapConn(ctx context.Context, cfg LDAPConfig) (*ldap.Conn, error) {
	address := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	tlsConfig := &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: cfg.InsecureSkipVerify}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if cfg.UseTLS {
		raw, err := (&tls.Dialer{NetDialer: &net.Dialer{Timeout: timeout}, Config: tlsConfig}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		conn := ldap.NewConn(raw, true)
		conn.Start()
		conn.SetTimeout(timeout)
		return conn, nil
	}
	raw, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	conn := ldap.NewConn(raw, false)
	conn.Start()
	if cfg.StartTLS {
		if err = conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, err
		}
	}
	conn.SetTimeout(timeout)
	return conn, nil
}

func ldapLoginUserFilter(cfg LDAPConfig, username string) string {
	userFilter := strings.ReplaceAll(cfg.UserFilter, "{username}", ldap.EscapeFilter(username))
	groupFilter := strings.TrimSpace(cfg.GroupFilter)
	if groupFilter == "" {
		return userFilter
	}
	if strings.HasPrefix(groupFilter, "(") {
		return "(&" + userFilter + groupFilter + ")"
	}
	return "(&" + userFilter + "(memberOf=" + ldap.EscapeFilter(groupFilter) + "))"
}

type ldapFilterSearchError struct {
	Source string
	Err    error
}

func (err ldapFilterSearchError) Error() string {
	return err.Source + " search failed: " + err.Err.Error()
}

func (err ldapFilterSearchError) Unwrap() error {
	return err.Err
}

func ldapTestUserFilter(cfg LDAPConfig) (string, string) {
	if strings.TrimSpace(cfg.GroupFilter) != "" {
		if strings.HasPrefix(strings.TrimSpace(cfg.GroupFilter), "(") {
			return strings.TrimSpace(cfg.GroupFilter), "用户组过滤器"
		}
		return "(memberOf=" + ldap.EscapeFilter(strings.TrimSpace(cfg.GroupFilter)) + ")", "用户组过滤器"
	}
	return strings.ReplaceAll(cfg.UserFilter, "{username}", "*"), "用户过滤器"
}

func countLDAPTestUsers(conn *ldap.Conn, cfg LDAPConfig) (int, error) {
	filter, source := ldapTestUserFilter(cfg)
	request := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		cfg.TimeoutSeconds,
		false,
		filter,
		[]string{"dn"},
		nil,
	)
	result, err := conn.SearchWithPaging(request, 500)
	if err != nil {
		return 0, ldapFilterSearchError{Source: source, Err: err}
	}
	return len(result.Entries), nil
}

func LDAPUserMessage(err error) string {
	if err == nil {
		return ""
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return "LDAP TLS 证书不受信任，请导入可信证书或勾选跳过证书校验"
	}
	var hostError x509.HostnameError
	if errors.As(err, &hostError) {
		return "LDAP TLS 证书域名与服务器地址不匹配，请检查证书或勾选跳过证书校验"
	}
	var certInvalidError x509.CertificateInvalidError
	if errors.As(err, &certInvalidError) {
		return "LDAP TLS 证书无效或已过期，请检查证书配置"
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		return "LDAP 服务连接超时，请检查服务器地址、端口和网络连通性"
	}
	if errors.Is(err, io.EOF) {
		return "LDAP 服务提前断开连接，请确认端口协议是否匹配"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "connection refused") {
		return "LDAP 服务拒绝连接，请检查端口是否开放"
	}
	if strings.Contains(message, "connection reset") {
		return "LDAP 连接被重置，请确认端口协议和 TLS 配置是否匹配"
	}
	if strings.Contains(message, "first record does not look like a tls handshake") {
		return "当前端口不是 LDAPS 服务，请改用 389 或关闭 LDAPS"
	}
	if strings.Contains(message, "unsupported protocol version") || strings.Contains(message, "protocol version 301") {
		return "LDAP TLS 版本过低，请启用 TLS 1.2+"
	}
	if strings.Contains(message, "start tls") || strings.Contains(message, "starttls") {
		return "LDAP 服务不支持 StartTLS 或 StartTLS 握手失败，请检查服务端配置"
	}
	if strings.Contains(message, "invalid credentials") || strings.Contains(message, "绑定账号或密码不正确") {
		return "LDAP 绑定账号或密码不正确"
	}
	var filterErr ldapFilterSearchError
	if errors.As(err, &filterErr) && strings.Contains(message, "filter compile error") {
		return filterErr.Source + "格式不正确，请填写完整 LDAP 过滤器"
	}
	if strings.Contains(message, "filter compile error") {
		return "LDAP 过滤器格式不正确，请检查用户过滤器或用户组过滤器"
	}
	return "认证服务连接测试失败：" + err.Error()
}
func (s *Service) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if strings.TrimSpace(token) == "" {
		return domain.User{}, repository.ErrNotFound
	}
	user, _, err := s.store.SessionUser(ctx, token, s.idleTTL)
	return user, err
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}
