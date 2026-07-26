package repository

import (
	"context"
	"errors"

	"certflow/backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListAudits(ctx context.Context, limit int) ([]domain.Audit, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, username, action, target, module, result, ip_address, detail, ts
		FROM audit_entries
		ORDER BY ts DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Audit{}
	for rows.Next() {
		var item domain.Audit
		if err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.Action,
			&item.Target,
			&item.Module,
			&item.Result,
			&item.IP,
			&item.Detail,
			&item.Timestamp,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	var username string
	var disabled bool
	if err := s.Pool.QueryRow(ctx, "SELECT username,disabled FROM users WHERE id=$1", id).Scan(&username, &disabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if username == defaultAdminUsername {
		return ErrDefaultAdminProtected
	}
	if !disabled {
		return ErrUserMustBeDisabled
	}
	result, err := s.Pool.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ChangePassword(ctx context.Context, id string, currentPassword string, newPassword string) error {
	var hash string
	var source string
	if err := s.Pool.QueryRow(ctx, "SELECT password_hash,source FROM users WHERE id=$1", id).Scan(&hash, &source); err != nil {
		return ErrNotFound
	}
	if source != "local" {
		return errors.New("LDAP 账号不能修改本地密码")
	}
	if err := VerifyPassword(hash, currentPassword); err != nil {
		return errors.New("当前密码不正确")
	}
	if VerifyPassword(hash, newPassword) == nil {
		return errors.New("新密码不能与当前密码相同")
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, "UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1", id, newHash)
	return err
}
