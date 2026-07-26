package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestDefaultAdminCredentials(t *testing.T) {
	if defaultAdminUsername != "admin" {
		t.Fatalf("default admin username = %q, want admin", defaultAdminUsername)
	}
	hash, err := HashPassword(defaultAdminPassword)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := VerifyPassword(hash, "123456"); err != nil {
		t.Fatalf("default admin password verification failed: %v", err)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	uniqueErr := &pgconn.PgError{Code: "23505", ConstraintName: "user_groups_name_key"}
	if !IsUniqueViolation(fmt.Errorf("insert user group: %w", uniqueErr)) {
		t.Fatal("wrapped unique violation was not detected")
	}
	if IsUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("foreign key violation was detected as a unique violation")
	}
	if IsUniqueViolation(errors.New("duplicate key value violates unique constraint")) {
		t.Fatal("plain error text was detected as a unique violation")
	}
}
