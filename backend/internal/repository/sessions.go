package repository

import (
	"context"
	"time"

	"certflow/backend/internal/domain"
)

// 用户登录会话记录的持久化访问：签发、校验、注销与过期清理

func (s *Store) CreateSession(ctx context.Context, token, userID, authProvider string, expires time.Time) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO sessions(token_hash,user_id,auth_provider,expires_at) VALUES($1,$2,$3,$4)", HashToken(token), userID, authProvider, expires)
	return err
}

func (s *Store) SessionUser(ctx context.Context, token string) (domain.User, time.Time, error) {
	var expires time.Time
	var id string
	var authProvider string
	err := s.Pool.QueryRow(ctx, "SELECT user_id::text,auth_provider,expires_at FROM sessions WHERE token_hash=$1", HashToken(token)).Scan(&id, &authProvider, &expires)
	if err != nil {
		return domain.User{}, time.Time{}, ErrNotFound
	}
	u, err := s.FindUserByID(ctx, id)
	if err != nil || u.Disabled || time.Now().After(expires) {
		return domain.User{}, time.Time{}, ErrNotFound
	}
	u.AuthenticationProvider = authProvider
	return u, expires, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM sessions WHERE token_hash=$1", HashToken(token))
	return err
}

// DeleteExpiredSessions 删除已过期的会话记录，登录签发令牌时惰性调用。
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at < now()")
	return err
}
