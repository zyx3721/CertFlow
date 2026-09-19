package repository

import (
	"context"
	"errors"

	"certflow/backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) FindUserByWecomBinding(ctx context.Context, wecomUserid string) (domain.User, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT u.id::text,u.username,u.display_name,u.email,u.password_hash,u.role,u.source,u.disabled,u.last_login_at,u.created_at,u.updated_at
		FROM user_wecom_bindings w
		JOIN users u ON u.id=w.user_id
		WHERE w.wecom_userid=$1
		LIMIT 1
	`, wecomUserid)
	u, _, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return s.AttachUserAccess(ctx, u)
}

func (s *Store) FindWecomBindingOwner(ctx context.Context, wecomUserid string) (string, error) {
	var userID string
	err := s.Pool.QueryRow(ctx, "SELECT user_id::text FROM user_wecom_bindings WHERE wecom_userid=$1", wecomUserid).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

func (s *Store) SaveWecomBinding(ctx context.Context, userID, wecomUserid string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO user_wecom_bindings(user_id,wecom_userid) VALUES($1,$2)
		ON CONFLICT (user_id) DO UPDATE SET wecom_userid=EXCLUDED.wecom_userid,bound_at=now()
	`, userID, wecomUserid)
	return err
}

func (s *Store) DeleteWecomBinding(ctx context.Context, userID string) (bool, error) {
	tag, err := s.Pool.Exec(ctx, "DELETE FROM user_wecom_bindings WHERE user_id=$1", userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) WecomBindingOfUser(ctx context.Context, userID string) (string, error) {
	var wecomUserid string
	err := s.Pool.QueryRow(ctx, "SELECT wecom_userid FROM user_wecom_bindings WHERE user_id=$1", userID).Scan(&wecomUserid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return wecomUserid, err
}
