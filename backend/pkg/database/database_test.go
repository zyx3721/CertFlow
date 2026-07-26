package database

import (
	"strings"
	"testing"
)

func TestInitialMigrationDoesNotRequirePostgreSQLExtensions(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, forbidden := range []string{"create extension", "pgcrypto", "gen_random_uuid"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("initial migration contains extension dependency %q", forbidden)
		}
	}
}

func TestInitialMigrationDocumentsTablesAndExcludesLegacyCaptchaTable(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	lines := strings.Split(string(content), "\n")
	for index, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "CREATE TABLE") {
			continue
		}
		if index == 0 || !strings.HasPrefix(strings.TrimSpace(lines[index-1]), "--") {
			t.Fatalf("table declaration on line %d is missing its preceding comment", index+1)
		}
	}
	if strings.Contains(string(content), "password_reset_captchas") {
		t.Fatal("initial migration must not create the unused password_reset_captchas table")
	}
}

func TestInitialMigrationIncludesRoleStateAndCAHierarchyCascade(t *testing.T) {
	initial, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}

	initialSQL := strings.ToLower(string(initial))
	if !strings.Contains(initialSQL, "disabled boolean not null default false") {
		t.Fatal("initial roles table does not define disabled state")
	}
	if !strings.Contains(initialSQL, "parent_id uuid references certificate_authorities(id) on delete cascade") {
		t.Fatal("initial CA hierarchy does not cascade deletes")
	}
}

func TestInitialMigrationAllowsCustomUserRoleKeys(t *testing.T) {
	initial, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	if strings.Contains(string(initial), "role TEXT NOT NULL CHECK") {
		t.Fatal("initial users.role column still restricts roles to built-in keys")
	}
}

func TestInitialMigrationSupportsMutualTLS(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"purpose text not null check (purpose in ('server', 'client', 'mtls'))",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("initial migration is missing %q", fragment)
		}
	}
}

func TestInitialMigrationUsesIdempotentDDL(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, table := range []string{
		"users",
		"sessions",
		"certificate_authorities",
		"ocsp_responders",
		"certificates",
		"certificate_revocation_entries",
		"workflows",
		"audit_entries",
		"system_settings",
		"roles",
		"user_roles",
		"user_groups",
		"user_group_members",
		"user_group_roles",
		"auth_provider_settings",
		"notification_channel_settings",
		"certificate_expiry_notifications",
		"password_reset_requests",
	} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Fatalf("initial migration table %q is not idempotent", table)
		}
	}
	for _, index := range []string{
		"idx_certificates_status",
		"idx_certificates_expiry",
		"idx_audit_entries_ts",
		"idx_password_reset_requests_user",
	} {
		if !strings.Contains(sql, "create index if not exists "+index) {
			t.Fatalf("initial migration index %q is not idempotent", index)
		}
	}
	if !strings.Contains(sql, "create unique index if not exists idx_certificates_renewed_from_id") {
		t.Fatal("initial migration auto renewal index is not idempotent")
	}
}

func TestInitialMigrationIncludesNotificationAndAutoRenewalSchema(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"renewed_from_id uuid references certificates(id) on delete set null",
		"references certificates(id) on delete set null",
		"create unique index if not exists idx_certificates_renewed_from_id",
		"approval_enabled boolean not null default false",
		"create table if not exists certificate_expiry_notifications",
		"channel_id text not null",
		"create table if not exists ocsp_responders",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("auto renewal migration is missing %q", fragment)
		}
	}
}

func TestInitialMigrationRetainsRevocationEntriesAfterCertificateDeletion(t *testing.T) {
	content, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"create table if not exists certificate_revocation_entries",
		"certificate_id uuid unique references certificates(id) on delete set null",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("initial migration is missing %q", fragment)
		}
	}
}

func TestInitialMigrationDefinesSessionAuthProvider(t *testing.T) {
	initial, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	if !strings.Contains(string(initial), "auth_provider TEXT NOT NULL DEFAULT 'local'") {
		t.Fatal("initial sessions schema does not define auth_provider")
	}
}
