package repository

import (
	"slices"
	"testing"

	"certflow/backend/internal/domain"
)

func TestNormalizePermissionsAddsImpliedReadPermission(t *testing.T) {
	permissions := normalizePermissions([]string{
		domain.PermissionCADownload,
		domain.PermissionCAAdd,
		domain.PermissionCADelete,
		domain.PermissionCertificateRequest,
		domain.PermissionSettingsUsersManage,
		"unknown.permission",
	})

	for _, expected := range []string{
		domain.PermissionCARead,
		domain.PermissionCADownload,
		domain.PermissionCAAdd,
		domain.PermissionCADelete,
		domain.PermissionCertificateRead,
		domain.PermissionCertificateRequest,
		domain.PermissionSettingsUsersRead,
		domain.PermissionSettingsUsersManage,
	} {
		if !slices.Contains(permissions, expected) {
			t.Fatalf("expected permission %q in %v", expected, permissions)
		}
	}
	if slices.Contains(permissions, "unknown.permission") {
		t.Fatalf("unexpected unknown permission in %v", permissions)
	}
}

func TestNormalizePermissionsAddsCertificateRequestDependencies(t *testing.T) {
	permissions := normalizePermissions([]string{domain.PermissionCertificateRequest})
	for _, expected := range []string{
		domain.PermissionCARead,
		domain.PermissionCertificateRead,
		domain.PermissionCertificateRequest,
	} {
		if !slices.Contains(permissions, expected) {
			t.Fatalf("expected permission %q in %v", expected, permissions)
		}
	}
}

func TestValidateManagedUserInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ManagedUserInput
		wantErr bool
	}{
		{name: "valid user", input: ManagedUserInput{Username: "operator", Email: "operator@example.com"}},
		{name: "missing email", input: ManagedUserInput{Username: "operator"}, wantErr: true},
		{name: "whitespace username", input: ManagedUserInput{Username: "bad user"}, wantErr: true},
		{name: "invalid email", input: ManagedUserInput{Username: "operator", Email: "not-an-email"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateManagedUserInput(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateManagedUserInput() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
