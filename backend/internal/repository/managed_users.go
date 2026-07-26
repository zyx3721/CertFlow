package repository

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"certflow/backend/internal/domain"

	"github.com/google/uuid"
)

type ManagedUserInput struct {
	Username    string
	DisplayName string
	Email       string
	Password    string
	RoleKeys    []string
	Disabled    bool
}

var (
	ErrDefaultAdminProtected        = errors.New("default admin protected")
	ErrManagedUserIdentityImmutable = errors.New("managed user identity immutable")
	ErrUserMustBeDisabled           = errors.New("user must be disabled before deletion")
)

func (s *Store) CreateManagedUser(ctx context.Context, input ManagedUserInput) (domain.User, error) {
	if err := validateManagedUserInput(input); err != nil {
		return domain.User{}, err
	}
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Email = strings.TrimSpace(input.Email)
	roleKeys := normalizeRoleKeys(input.RoleKeys)
	if input.Username == "" {
		return domain.User{}, errors.New("用户名不能为空")
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if len(input.Password) < 6 || len(input.Password) > 256 {
		return domain.User{}, errors.New("密码长度必须在 6 到 256 位之间")
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}
	id := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,email,password_hash,role,source,disabled)
		VALUES($1,$2,$3,$4,$5,$6,'local',$7)
	`, id, input.Username, input.DisplayName, input.Email, hash, roleKeys[0], input.Disabled)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.SetUserRoles(ctx, id, roleKeys); err != nil {
		return domain.User{}, err
	}
	return s.FindUserByID(ctx, id)
}

func (s *Store) UpdateManagedUser(ctx context.Context, id string, input ManagedUserInput) (domain.User, error) {
	if err := validateManagedUserInput(input); err != nil {
		return domain.User{}, err
	}
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Email = strings.TrimSpace(input.Email)
	roleKeys := normalizeRoleKeys(input.RoleKeys)
	if input.Username == "" {
		return domain.User{}, errors.New("用户名不能为空")
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	var currentUsername string
	if err := s.Pool.QueryRow(ctx, "SELECT username FROM users WHERE id=$1", id).Scan(&currentUsername); err != nil {
		return domain.User{}, ErrNotFound
	}
	if currentUsername == defaultAdminUsername && (input.Username != defaultAdminUsername || input.Disabled) {
		return domain.User{}, ErrDefaultAdminProtected
	}
	if currentUsername != input.Username {
		return domain.User{}, ErrManagedUserIdentityImmutable
	}
	query := "UPDATE users SET username=$2,display_name=$3,email=$4,role=$5,source='local',disabled=$6,updated_at=now() WHERE id=$1"
	args := []any{id, input.Username, input.DisplayName, input.Email, roleKeys[0], input.Disabled}
	if input.Password != "" {
		if len(input.Password) < 6 || len(input.Password) > 256 {
			return domain.User{}, errors.New("密码长度必须在 6 到 256 位之间")
		}
		hash, err := HashPassword(input.Password)
		if err != nil {
			return domain.User{}, err
		}
		query = "UPDATE users SET username=$2,display_name=$3,email=$4,role=$5,source='local',disabled=$6,password_hash=$7,updated_at=now() WHERE id=$1"
		args = append(args, hash)
	}
	result, err := s.Pool.Exec(ctx, query, args...)
	if err != nil {
		return domain.User{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.User{}, ErrNotFound
	}
	if err := s.SetUserRoles(ctx, id, roleKeys); err != nil {
		return domain.User{}, err
	}
	return s.FindUserByID(ctx, id)
}

func validateManagedUserInput(input ManagedUserInput) error {
	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	if username == "" || len(username) > 128 || strings.ContainsAny(username, "\r\n\t ") {
		return errors.New("用户名不能为空、不能包含空白且长度不能超过 128 位")
	}
	if email == "" {
		return errors.New("邮箱不能为空")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return errors.New("邮箱格式不正确")
	}
	return nil
}
