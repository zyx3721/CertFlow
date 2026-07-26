package router

import (
	"testing"
)

func TestAuthProvidersResponse(t *testing.T) {
	tests := []struct {
		name                 string
		items                []map[string]any
		passwordResetEnabled bool
	}{
		{name: "all disabled", items: nil, passwordResetEnabled: false},
		{name: "LDAP and password reset enabled", items: []map[string]any{{"id": "ldap", "type": "ldap", "name": "AD/LDAP", "enabled": true}}, passwordResetEnabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := authProvidersResponse(test.items, test.passwordResetEnabled)
			if body["total"] != len(test.items) {
				t.Fatalf("expected %d providers, got %v", len(test.items), body["total"])
			}
			if body["passwordResetEnabled"] != test.passwordResetEnabled {
				t.Fatalf("expected password reset enabled=%v, got %v", test.passwordResetEnabled, body["passwordResetEnabled"])
			}
		})
	}
}
