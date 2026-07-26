package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"certflow/backend/internal/domain"
)

func TestDomainUserUsesCamelCaseJSON(t *testing.T) {
	body, err := json.Marshal(domain.User{
		ID:          "user-1",
		Username:    "admin",
		DisplayName: "管理员",
		Source:      "local",
		Roles:       []domain.Role{},
		DirectRoles: []domain.Role{},
		Permissions: domain.PermissionsForRole("user"),
	})
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}
	text := string(body)
	for _, key := range []string{`"displayName"`, `"permissions"`, `"roles":[]`, `"directRoles":[]`, `"source"`} {
		if !strings.Contains(text, key) {
			t.Fatalf("expected key %s in %s", key, text)
		}
	}
	if strings.Contains(text, `"DisplayName"`) || strings.Contains(text, `"Permissions"`) {
		t.Fatalf("unexpected PascalCase JSON keys in %s", text)
	}
}

func TestLoginSessionContractUsesCamelCaseExpiry(t *testing.T) {
	response := httptest.NewRecorder()
	write(response, http.StatusOK, map[string]any{
		"token":     "test-token",
		"expiresAt": time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
		"user":      domain.User{Username: "admin", Source: "local"},
	})
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["expiresAt"]; !ok {
		t.Fatalf("expiresAt missing from response: %+v", body)
	}
	if _, ok := body["expires_at"]; ok {
		t.Fatalf("legacy expires_at unexpectedly present: %+v", body)
	}
}
