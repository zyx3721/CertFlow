package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"certflow/backend/internal/repository"
)

func (s *Service) PublicProviders(ctx context.Context) ([]map[string]any, bool, error) {
	items := []map[string]any{}
	ldap, err := s.store.AuthProviderSetting(ctx, "ldap")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, false, err
	}
	if err == nil && ldap.Enabled {
		items = append(items, map[string]any{"id": "ldap", "type": "ldap", "name": ldap.Name, "enabled": true})
	}
	email, err := s.store.NotificationChannelSetting(ctx, "email")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, false, err
	}
	return items, err == nil && email.PasswordResetEnabled, nil
}

func (s *Service) LDAPSetting(ctx context.Context) (repository.AuthProviderSetting, error) {
	item, err := s.store.AuthProviderSetting(ctx, "ldap")
	if err != nil {
		return item, err
	}
	item.Config["hasBindPassword"] = item.SecretCiphertext != ""
	return item, nil
}

func (s *Service) SaveLDAPSetting(ctx context.Context, name string, enabled bool, values map[string]any) (repository.AuthProviderSetting, error) {
	current, err := s.store.AuthProviderSetting(ctx, "ldap")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return current, err
	}
	values = sanitizeLDAPSettingValues(values)
	password := strings.TrimSpace(stringValue(values, "bindPassword"))
	delete(values, "bindPassword")
	delete(values, "hasBindPassword")
	if password != "" {
		current.SecretCiphertext, err = s.box.Seal(password)
		if err != nil {
			return current, err
		}
	}
	current.ID = "ldap"
	current.Type = "ldap"
	current.Name = strings.TrimSpace(name)
	if current.Name == "" {
		current.Name = "AD/LDAP"
	}
	current.Enabled = enabled
	current.Config = values
	if _, err := ldapConfigFromSetting(current, s.box.Open); enabled && err != nil {
		return current, err
	}
	if err := s.store.SaveAuthProviderSetting(ctx, current); err != nil {
		return current, err
	}
	return s.LDAPSetting(ctx)
}

func (s *Service) ClearLDAPSetting(ctx context.Context) (repository.AuthProviderSetting, error) {
	item, err := s.store.AuthProviderSetting(ctx, "ldap")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return item, err
	}
	item.ID = "ldap"
	item.Type = "ldap"
	item.Name = "AD/LDAP"
	item.Enabled = false
	item.Config = map[string]any{}
	item.SecretCiphertext = ""
	if err := s.store.SaveAuthProviderSetting(ctx, item); err != nil {
		return item, err
	}
	return s.LDAPSetting(ctx)
}

func (s *Service) TestLDAPSetting(ctx context.Context) (int, error) {
	setting, err := s.store.AuthProviderSetting(ctx, "ldap")
	if err != nil {
		return 0, err
	}
	cfg, err := ldapConfigFromSetting(setting, s.box.Open)
	if err != nil {
		return 0, err
	}
	conn, err := ldapConn(ctx, cfg)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if cfg.BindDN != "" && conn.Bind(cfg.BindDN, cfg.BindPassword) != nil {
		return 0, errors.New("LDAP 绑定账号或密码不正确")
	}
	matchedUsers, err := countLDAPTestUsers(conn, cfg)
	if err != nil {
		return 0, err
	}
	return matchedUsers, nil
}

func sanitizeLDAPSettingValues(values map[string]any) map[string]any {
	cleaned := make(map[string]any, len(values))
	for key, value := range values {
		if key == "hasBindPassword" || key == "defaultRole" || key == "adminGroupDN" {
			continue
		}
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			cleaned[key] = text
			continue
		}
		if value != nil {
			cleaned[key] = value
		}
	}
	return cleaned
}

type LDAPConfig struct {
	Enabled                                                                                                 bool
	Host, BaseDN, BindDN, BindPassword, UserFilter, UsernameAttribute, DisplayNameAttribute, EmailAttribute string
	GroupFilter                                                                                             string
	Port, TimeoutSeconds                                                                                    int
	UseTLS, StartTLS, InsecureSkipVerify                                                                    bool
}

func (s *Service) runtimeLDAPConfig(ctx context.Context) (LDAPConfig, error) {
	setting, err := s.store.AuthProviderSetting(ctx, "ldap")
	if errors.Is(err, repository.ErrNotFound) {
		return LDAPConfig{}, nil
	}
	if err != nil {
		return LDAPConfig{}, err
	}
	return ldapConfigFromSetting(setting, s.box.Open)
}

func ldapConfigFromSetting(setting repository.AuthProviderSetting, open func(string) (string, error)) (LDAPConfig, error) {
	values := setting.Config
	cfg := LDAPConfig{
		Enabled:              setting.Enabled,
		Host:                 strings.TrimSpace(stringValue(values, "host")),
		Port:                 intValue(values, "port", 0),
		BaseDN:               strings.TrimSpace(stringValue(values, "baseDN")),
		BindDN:               strings.TrimSpace(stringValue(values, "bindDN")),
		UseTLS:               boolValue(values, "useTLS"),
		StartTLS:             boolValue(values, "startTLS"),
		InsecureSkipVerify:   boolValue(values, "insecureSkipVerify"),
		UserFilter:           strings.TrimSpace(stringValue(values, "userFilter")),
		UsernameAttribute:    "sAMAccountName",
		DisplayNameAttribute: "displayName",
		EmailAttribute:       "mail",
		TimeoutSeconds:       intValue(values, "timeoutSeconds", 8),
		GroupFilter:          strings.TrimSpace(stringValue(values, "groupFilter")),
	}
	if setting.SecretCiphertext != "" {
		password, err := open(setting.SecretCiphertext)
		if err != nil {
			return cfg, err
		}
		cfg.BindPassword = password
	}
	if cfg.TimeoutSeconds < 1 || cfg.TimeoutSeconds > 30 {
		cfg.TimeoutSeconds = 8
	}
	if cfg.Enabled {
		if cfg.Host == "" {
			return cfg, errors.New("服务器地址不能为空")
		}
		if cfg.BaseDN == "" {
			return cfg, errors.New("Base DN 不能为空")
		}
		if cfg.Port < 1 || cfg.Port > 65535 {
			return cfg, errors.New("端口不能为空")
		}
		if cfg.UserFilter == "" {
			return cfg, errors.New("用户过滤器不能为空")
		}
		if cfg.BindDN == "" {
			return cfg, errors.New("绑定 DN 不能为空")
		}
		if cfg.BindPassword == "" {
			return cfg, errors.New("绑定密码不能为空")
		}
	}
	if cfg.Enabled && cfg.UseTLS && cfg.StartTLS {
		return cfg, errors.New("LDAPS 与 StartTLS 不能同时启用")
	}
	return cfg, nil
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func boolValue(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func intValue(values map[string]any, key string, fallback int) int {
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
		return fallback
	default:
		return fallback
	}
}
