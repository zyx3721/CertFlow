package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type PasswordResetRequest struct {
	TokenHash     string
	UserID        string
	CodeHash      string
	CodeExpiresAt *time.Time
	CodeSentAt    *time.Time
	UsedAt        *time.Time
	ExpiresAt     time.Time
}

func (s *Store) CreatePasswordResetRequest(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO password_reset_requests(token_hash,user_id,expires_at) VALUES($1,$2,$3)", tokenHash, userID, expiresAt)
	return err
}

func (s *Store) PasswordResetRequest(ctx context.Context, tokenHash string) (PasswordResetRequest, error) {
	var item PasswordResetRequest
	err := s.Pool.QueryRow(ctx, `
		SELECT token_hash,user_id::text,code_hash,code_expires_at,code_sent_at,used_at,expires_at
		FROM password_reset_requests WHERE token_hash=$1
	`, tokenHash).Scan(&item.TokenHash, &item.UserID, &item.CodeHash, &item.CodeExpiresAt, &item.CodeSentAt, &item.UsedAt, &item.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}

func (s *Store) SetPasswordResetCode(ctx context.Context, tokenHash string, codeHash string, expiresAt time.Time) error {
	result, err := s.Pool.Exec(ctx, `
		UPDATE password_reset_requests
		SET code_hash=$2,code_expires_at=$3,code_sent_at=now()
		WHERE token_hash=$1 AND used_at IS NULL AND expires_at>now()
	`, tokenHash, codeHash, expiresAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountRecentPasswordResetCodes(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM password_reset_requests
		WHERE user_id=$1 AND code_sent_at IS NOT NULL AND code_sent_at >= $2
	`, userID, since).Scan(&count)
	return count, err
}

func (s *Store) CompletePasswordReset(ctx context.Context, tokenHash string, userID string, passwordHash string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, "UPDATE password_reset_requests SET used_at=now() WHERE token_hash=$1 AND user_id=$2 AND used_at IS NULL", tokenHash, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err = tx.Exec(ctx, "UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1 AND source='local'", userID, passwordHash); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM sessions WHERE user_id=$1", userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
