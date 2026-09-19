package auth

import (
	"context"
	"errors"
	"strings"

	"certflow/backend/internal/repository"
)

// WecomSettingView 是管理端读取到的企业微信认证配置视图，密钥只回显是否已配置。
type WecomSettingView struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Enabled   bool           `json:"enabled"`
	Config    map[string]any `json:"config"`
	UpdatedAt string         `json:"updatedAt"`
}

const wecomSecretCipherKey = "ssoAppSecretCipher"

// WecomSetting 读取企业微信认证配置，密钥字段不回显原文。
func (s *Service) WecomSetting(ctx context.Context) (WecomSettingView, error) {
	setting, err := s.store.AuthProviderSetting(ctx, "wecom")
	if errors.Is(err, repository.ErrNotFound) {
		return WecomSettingView{ID: "wecom", Type: "wecom", Name: "企业微信", Config: wecomSettingConfigView(nil, false, false)}, nil
	}
	if err != nil {
		return WecomSettingView{}, err
	}
	hasSecret := setting.SecretCiphertext != ""
	hasSSOSecret := strings.TrimSpace(stringValue(setting.Config, wecomSecretCipherKey)) != ""
	return WecomSettingView{
		ID:        "wecom",
		Type:      "wecom",
		Name:      displayName(setting.Name, "企业微信"),
		Enabled:   setting.Enabled,
		Config:    wecomSettingConfigView(setting.Config, hasSecret, hasSSOSecret),
		UpdatedAt: setting.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func wecomSettingConfigView(values map[string]any, hasSecret, hasSSOSecret bool) map[string]any {
	if values == nil {
		values = map[string]any{}
	}
	view := map[string]any{
		"mode":            displayValue(values, "mode", WecomModeDirect),
		"corpid":          stringValue(values, "corpid"),
		"agentid":         stringValue(values, "agentid"),
		"redirectPrefix":  stringValue(values, "redirectPrefix"),
		"ssoBaseUrl":      stringValue(values, "ssoBaseUrl"),
		"ssoAppID":        stringValue(values, "ssoAppID"),
		"hasSecret":       hasSecret,
		"hasSsoAppSecret": hasSSOSecret,
	}
	return view
}

// SaveWecomSetting 保存企业微信认证配置；secret 留空表示沿用旧值，仅 clearConfig 时清空。
func (s *Service) SaveWecomSetting(ctx context.Context, name string, enabled bool, values map[string]any) (WecomSettingView, error) {
	current, err := s.store.AuthProviderSetting(ctx, "wecom")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return WecomSettingView{}, err
	}

	config := map[string]any{
		"mode":           normalizeWecomMode(values["mode"]),
		"corpid":         trimmedValue(values, "corpid"),
		"agentid":        trimmedValue(values, "agentid"),
		"redirectPrefix": strings.TrimSuffix(trimmedValue(values, "redirectPrefix"), "/"),
		"ssoBaseUrl":     strings.TrimSuffix(trimmedValue(values, "ssoBaseUrl"), "/"),
		"ssoAppID":       trimmedValue(values, "ssoAppID"),
	}
	if secret := trimmedValue(values, "secret"); secret != "" {
		ciphertext, sealErr := s.box.Seal(secret)
		if sealErr != nil {
			return WecomSettingView{}, sealErr
		}
		current.SecretCiphertext = ciphertext
	}
	if cipher := stringValue(current.Config, wecomSecretCipherKey); cipher != "" {
		config[wecomSecretCipherKey] = cipher
	}
	if appSecret := trimmedValue(values, "ssoAppSecret"); appSecret != "" {
		ciphertext, sealErr := s.box.Seal(appSecret)
		if sealErr != nil {
			return WecomSettingView{}, sealErr
		}
		config[wecomSecretCipherKey] = ciphertext
	}

	setting := repository.AuthProviderSetting{
		ID:               "wecom",
		Type:             "wecom",
		Name:             displayName(name, "企业微信"),
		Enabled:          enabled,
		Config:           config,
		SecretCiphertext: current.SecretCiphertext,
	}
	if err := validateWecomSetting(setting, s.box.Open); enabled && err != nil {
		return WecomSettingView{}, err
	}
	if err := s.store.SaveAuthProviderSetting(ctx, setting); err != nil {
		return WecomSettingView{}, err
	}
	return s.WecomSetting(ctx)
}

// ClearWecomSetting 清空企业微信认证配置并停用该登录方式。
func (s *Service) ClearWecomSetting(ctx context.Context) (WecomSettingView, error) {
	setting := repository.AuthProviderSetting{ID: "wecom", Type: "wecom", Name: "企业微信", Enabled: false, Config: map[string]any{}, SecretCiphertext: ""}
	if err := s.store.SaveAuthProviderSetting(ctx, setting); err != nil {
		return WecomSettingView{}, err
	}
	return s.WecomSetting(ctx)
}

func validateWecomSetting(setting repository.AuthProviderSetting, open func(string) (string, error)) error {
	cfg, err := runtimeWecomConfigFromSetting(setting, open)
	if err != nil {
		return err
	}
	return cfg.validate()
}

func runtimeWecomConfigFromSetting(setting repository.AuthProviderSetting, open func(string) (string, error)) (WecomConfig, error) {
	values := setting.Config
	cfg := WecomConfig{
		Enabled:        setting.Enabled,
		Mode:           normalizeWecomMode(values["mode"]),
		CorpID:         strings.TrimSpace(stringValue(values, "corpid")),
		AgentID:        strings.TrimSpace(stringValue(values, "agentid")),
		RedirectPrefix: strings.TrimSpace(stringValue(values, "redirectPrefix")),
		SSOBaseURL:     strings.TrimSpace(stringValue(values, "ssoBaseUrl")),
		SSOAppID:       strings.TrimSpace(stringValue(values, "ssoAppID")),
	}
	if setting.SecretCiphertext != "" {
		secret, err := open(setting.SecretCiphertext)
		if err != nil {
			return cfg, err
		}
		cfg.Secret = secret
	}
	if cipher := stringValue(values, wecomSecretCipherKey); cipher != "" {
		secret, err := open(cipher)
		if err != nil {
			return cfg, err
		}
		cfg.SSOAppSecret = secret
	}
	return cfg, nil
}

func normalizeWecomMode(value any) string {
	switch mode := value.(type) {
	case string:
		if strings.TrimSpace(mode) == WecomModeSSO {
			return WecomModeSSO
		}
	case float64:
		return WecomModeDirect
	case bool:
		return WecomModeDirect
	}
	return WecomModeDirect
}

func trimmedValue(values map[string]any, key string) string {
	return strings.TrimSpace(stringValue(values, key))
}

func displayName(name, fallback string) string {
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return strings.TrimSpace(name)
}

func displayValue(values map[string]any, key, fallback string) string {
	if value := strings.TrimSpace(stringValue(values, key)); value != "" {
		return value
	}
	return fallback
}
