package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"certflow/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("record not found")

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "123456"
)

type Store struct{ Pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
func scanUser(row pgx.Row) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &hash, &u.Role, &u.Source, &u.Disabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, hash, err
}
func (s *Store) EnsureDefaultAdmin(ctx context.Context) error {
	var count int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count > 0 {
		return err
	}
	hash, err := HashPassword(defaultAdminPassword)
	if err != nil {
		return err
	}
	id := uuid.NewString()
	if _, err = s.Pool.Exec(ctx, "INSERT INTO users(id,username,display_name,password_hash,role) VALUES($1,$2,$2,$3,'admin')", id, defaultAdminUsername, hash); err != nil {
		return err
	}
	return s.SetUserRoles(ctx, id, []string{"admin"})
}
func (s *Store) FindUser(ctx context.Context, username string) (domain.User, string, error) {
	u, h, err := scanUser(s.Pool.QueryRow(ctx, "SELECT id::text,username,display_name,email,password_hash,role,source,disabled,last_login_at,created_at,updated_at FROM users WHERE username=$1", username))
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	if err != nil {
		return u, h, err
	}
	u, err = s.AttachUserAccess(ctx, u)
	return u, h, err
}
func (s *Store) FindUserByID(ctx context.Context, id string) (domain.User, error) {
	u, _, err := scanUser(s.Pool.QueryRow(ctx, "SELECT id::text,username,display_name,email,password_hash,role,source,disabled,last_login_at,created_at,updated_at FROM users WHERE id=$1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	if err != nil {
		return u, err
	}
	return s.AttachUserAccess(ctx, u)
}
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, "SELECT id::text,username,display_name,email,password_hash,role,source,disabled,last_login_at,created_at,updated_at FROM users ORDER BY created_at,username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.User{}
	for rows.Next() {
		user, _, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		user, err = s.AttachUserAccess(ctx, user)
		if err != nil {
			return nil, err
		}
		items = append(items, user)
	}
	return items, rows.Err()
}
func (s *Store) SetUserDisabled(ctx context.Context, id string, disabled bool) error {
	result, err := s.Pool.Exec(ctx, "UPDATE users SET disabled=$2,updated_at=now() WHERE id=$1", id, disabled)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) RecordUserLogin(ctx context.Context, userID string) error {
	result, err := s.Pool.Exec(ctx, "UPDATE users SET last_login_at=now(),updated_at=now() WHERE id=$1", userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) Audit(ctx context.Context, user domain.User, action, target, module, result, ip, detail string) {
	_, _ = s.Pool.Exec(ctx, "INSERT INTO audit_entries(id,user_id,username,action,target,module,result,ip_address,detail) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9)", uuid.NewString(), user.ID, user.Username, action, target, module, result, ip, detail)
}
