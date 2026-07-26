package auth

import (
	"errors"
	"strings"
	"testing"

	"certflow/backend/internal/repository"
)

func TestLDAPConfigRequiresPortWhenEnabled(t *testing.T) {
	_, err := ldapConfigFromSetting(repository.AuthProviderSetting{
		Enabled: true,
		Config: map[string]any{
			"host":       "ldap.example.com",
			"baseDN":     "dc=example,dc=com",
			"userFilter": "(uid={username})",
			"bindDN":     "cn=readonly,dc=example,dc=com",
		},
		SecretCiphertext: "configured",
	}, func(string) (string, error) { return "secret", nil })
	if err == nil || !strings.Contains(err.Error(), "端口不能为空") {
		t.Fatalf("expected missing port error, got %v", err)
	}
}

func TestLDAPConfigRejectsConflictingTransportSettings(t *testing.T) {
	_, err := ldapConfigFromSetting(repository.AuthProviderSetting{
		Enabled: true,
		Config: map[string]any{
			"host":       "ldap.example.com",
			"port":       636,
			"baseDN":     "dc=example,dc=com",
			"userFilter": "(uid={username})",
			"bindDN":     "cn=readonly,dc=example,dc=com",
			"useTLS":     true,
			"startTLS":   true,
		},
		SecretCiphertext: "configured",
	}, func(string) (string, error) { return "secret", nil })
	if err == nil || !strings.Contains(err.Error(), "不能同时启用") {
		t.Fatalf("expected conflicting TLS settings error, got %v", err)
	}
}

func TestSanitizeLDAPSettingValuesPreservesOptionalSwitches(t *testing.T) {
	values := sanitizeLDAPSettingValues(map[string]any{
		"useTLS":             false,
		"startTLS":           true,
		"insecureSkipVerify": true,
		"timeoutSeconds":     8,
		"groupFilter":        "  cn=ops,dc=example,dc=com  ",
		"defaultRole":        "admin",
		"adminGroupDN":       "cn=admins,dc=example,dc=com",
	})
	if values["useTLS"] != false || values["startTLS"] != true || values["insecureSkipVerify"] != true {
		t.Fatalf("optional LDAP switches were not preserved: %#v", values)
	}
	if values["groupFilter"] != "cn=ops,dc=example,dc=com" {
		t.Fatalf("unexpected normalized group filter: %#v", values["groupFilter"])
	}
	if _, ok := values["defaultRole"]; ok {
		t.Fatalf("legacy defaultRole must be removed: %#v", values)
	}
}

func TestLDAPLoginUserFilterIncludesGroupFilter(t *testing.T) {
	filter := ldapLoginUserFilter(LDAPConfig{
		UserFilter:  "(uid={username})",
		GroupFilter: "cn=operators,dc=example,dc=com",
	}, "alice*(test)")
	want := "(&(uid=alice\\2a\\28test\\29)(memberOf=cn=operators,dc=example,dc=com))"
	if filter != want {
		t.Fatalf("unexpected LDAP filter: %q", filter)
	}
}

func TestLDAPTestUserFilterUsesGroupFilterWhenConfigured(t *testing.T) {
	filter, source := ldapTestUserFilter(LDAPConfig{
		UserFilter:  "(uid={username})",
		GroupFilter: "cn=operators,dc=example,dc=com",
	})
	if source != "用户组过滤器" || filter != "(memberOf=cn=operators,dc=example,dc=com)" {
		t.Fatalf("unexpected LDAP test filter: source=%q filter=%q", source, filter)
	}
}

func TestLDAPUserMessageExplainsConnectionRefusal(t *testing.T) {
	message := LDAPUserMessage(errors.New("connection refused"))
	if message != "LDAP 服务拒绝连接，请检查端口是否开放" {
		t.Fatalf("unexpected LDAP user message: %q", message)
	}
}
