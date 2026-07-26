package router

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
	authsvc "certflow/backend/internal/service/auth"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPublicSettingsPayloadUsesConfiguredValuesAndDefaults(t *testing.T) {
	payload := publicSettingsPayload(map[string]any{
		"siteName":               "  CertFlow Test  ",
		"loginName":              "登录品牌",
		"appName":                "控制台品牌",
		"appSubtitle":            "PKI Test",
		"iconData":               "/custom.svg",
		"expiryNotificationDays": float64(21),
	})
	if payload["siteName"] != "CertFlow Test" || payload["iconData"] != "/custom.svg" || payload["expiryNotificationDays"] != 21 {
		t.Fatalf("payload = %#v", payload)
	}
	defaults := publicSettingsPayload(nil)
	if defaults["siteName"] != "CertFlow" || defaults["iconData"] != "/favicon.svg" || defaults["appSubtitle"] != "PKI Control Plane" || defaults["expiryNotificationDays"] != 15 {
		t.Fatalf("defaults = %#v", defaults)
	}
}

func TestDecodeManagedUserRejectsAccountSource(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/settings/users", bytes.NewBufferString(`{
		"username":"directory-user",
		"email":"directory-user@example.com",
		"password":"123456789012",
		"source":"ldap",
		"roleKeys":["viewer"]
	}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := decodeManagedUser(req, true); err == nil {
		t.Fatal("managed-user request unexpectedly accepted source")
	}
}

func TestDecodeManagedUserAcceptsSixCharacterPassword(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/settings/users", bytes.NewBufferString(`{
		"username":"operator",
		"email":"operator@example.com",
		"password":"123456",
		"roleKeys":[]
	}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	input, err := decodeManagedUser(req, true)
	if err != nil {
		t.Fatalf("decode managed user: %v", err)
	}
	if input.DisplayName != "operator" || len(input.RoleKeys) != 1 || input.RoleKeys[0] != "viewer" {
		t.Fatalf("input = %#v", input)
	}
}

func TestLoginFailureMessageForUnprovisionedLDAPUser(t *testing.T) {
	if message := loginFailureMessage(authsvc.ErrUserNotProvisioned); message != "用户未在平台中启用" {
		t.Fatalf("message = %q", message)
	}
}

func TestAuthenticationAuditDetailUsesUserSource(t *testing.T) {
	tests := []struct {
		name     string
		username string
		source   string
		action   string
		want     string
	}{
		{name: "local login", username: "alice", source: "local", action: "登录", want: "本地用户 alice 登录成功"},
		{name: "LDAP logout", username: "bob", source: "ldap", action: "注销", want: "LDAP 用户 bob 注销成功"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := domain.User{Username: test.username, AuthenticationProvider: test.source}
			if got := authenticationAuditDetail(user, test.action); got != test.want {
				t.Fatalf("detail = %q, want %q", got, test.want)
			}
		})
	}
}

func TestConfigurationSaveErrorsReturnFriendlyConflictWithoutDatabaseDetails(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		message    string
		writeError func(http.ResponseWriter, error)
	}{
		{name: "managed user", constraint: "users_username_key", message: "用户名已存在", writeError: writeManagedUserSaveError},
		{name: "user role", constraint: "roles_key_key", message: "角色标识已存在", writeError: writeRoleSaveError},
		{name: "user group", constraint: "user_groups_name_key", message: "用户群组名称已存在", writeError: writeUserGroupSaveError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			databaseErr := &pgconn.PgError{
				Code:           "23505",
				ConstraintName: test.constraint,
				Detail:         "duplicate key value violates unique constraint",
			}

			test.writeError(recorder, databaseErr)

			if recorder.Code != http.StatusConflict {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, test.message) {
				t.Fatalf("body = %q", body)
			}
			if strings.Contains(body, databaseErr.Detail) || strings.Contains(body, databaseErr.ConstraintName) {
				t.Fatalf("response exposed database details: %q", body)
			}
		})
	}
}

func TestConfigurationSaveErrorsMapKnownAndUnknownFailures(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		message    string
		writeError func(http.ResponseWriter, error)
	}{
		{name: "missing selected role for user", err: repository.ErrRoleNotFound, status: http.StatusBadRequest, message: "选择的角色不存在", writeError: writeManagedUserSaveError},
		{name: "missing selected role for group", err: repository.ErrRoleNotFound, status: http.StatusBadRequest, message: "选择的角色不存在", writeError: writeUserGroupSaveError},
		{name: "protected default admin", err: repository.ErrDefaultAdminProtected, status: http.StatusBadRequest, message: "默认管理员账号不能改名或禁用", writeError: writeManagedUserSaveError},
		{name: "managed user identity immutable", err: repository.ErrManagedUserIdentityImmutable, status: http.StatusBadRequest, message: "用户创建后用户名不可修改", writeError: writeManagedUserSaveError},
		{name: "role identity immutable", err: repository.ErrRoleIdentityImmutable, status: http.StatusBadRequest, message: "用户角色创建后标识和名称不可修改", writeError: writeRoleSaveError},
		{name: "user group name immutable", err: repository.ErrUserGroupNameImmutable, status: http.StatusBadRequest, message: "用户群组创建后名称不可修改", writeError: writeUserGroupSaveError},
		{name: "unknown user storage failure", err: errors.New("database unavailable"), status: http.StatusInternalServerError, message: "保存用户失败", writeError: writeManagedUserSaveError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.writeError(recorder, test.err)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, test.message) {
				t.Fatalf("body = %q", body)
			}
			if strings.Contains(body, test.err.Error()) && test.err.Error() != test.message {
				t.Fatalf("response exposed internal error: %q", body)
			}
		})
	}
}

func TestWriteDeleteUserErrorRequiresDisabledUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeDeleteUserError(recorder, repository.ErrUserMustBeDisabled)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "请先禁用用户再删除") {
		t.Fatalf("body = %q", body)
	}
}

func TestWriteRoleDeleteErrorRequiresDisabledRole(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRoleDeleteError(recorder, repository.ErrRoleMustBeDisabled)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "请先禁用用户角色再删除") {
		t.Fatalf("body = %q", body)
	}
}
