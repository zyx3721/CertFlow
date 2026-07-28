package repository

import (
	"context"
	"errors"
	"sort"
	"strings"

	"certflow/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var BuiltinPermissions = []domain.Permission{
	{Key: domain.PermissionDashboardRead, Name: "查看仪表盘", Description: "查看 PKI 统计、趋势和近期活动", Category: "仪表盘"},
	{Key: domain.PermissionCARead, Name: "查看 CA", Description: "查看 CA 层级和公开证书", Category: "CA 管理"},
	{Key: domain.PermissionCADownload, Name: "下载 CA", Description: "下载 CA 证书文件", Category: "CA 管理", ImpliedReadPermission: domain.PermissionCARead},
	{Key: domain.PermissionCAAdd, Name: "添加 CA", Description: "创建或导入根、中间或签发 CA", Category: "CA 管理", ImpliedReadPermission: domain.PermissionCARead},
	{Key: domain.PermissionCADelete, Name: "删除 CA", Description: "删除未关联证书的 CA", Category: "CA 管理", ImpliedReadPermission: domain.PermissionCARead},
	{Key: domain.PermissionCertificateRead, Name: "查看证书", Description: "查看证书列表和详情", Category: "证书管理"},
	{Key: domain.PermissionCertificateRequest, Name: "申请证书", Description: "提交证书申请，需要读取可用 CA", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead, ImpliedPermissions: []string{domain.PermissionCARead}},
	{Key: domain.PermissionCertificateDownload, Name: "下载证书", Description: "下载授权范围内的证书材料", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead},
	{Key: domain.PermissionCertificateVerify, Name: "校验证书", Description: "校验证书状态和证书链", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead},
	{Key: domain.PermissionCertificateManage, Name: "管理证书", Description: "导出证书数据", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead},
	{Key: domain.PermissionCertificateRevoke, Name: "撤销证书", Description: "撤销有效证书", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead},
	{Key: domain.PermissionCertificateDelete, Name: "删除证书", Description: "删除非有效状态证书并导出证书数据", Category: "证书管理", ImpliedReadPermission: domain.PermissionCertificateRead},
	{Key: domain.PermissionWorkflowRead, Name: "查看审批", Description: "查看证书申请审批列表", Category: "审批管理"},
	{Key: domain.PermissionWorkflowApprove, Name: "准驳审批", Description: "通过或驳回待审批证书申请", Category: "审批管理", ImpliedReadPermission: domain.PermissionWorkflowRead},
	{Key: domain.PermissionWorkflowDelete, Name: "删除审批", Description: "删除待审批证书申请", Category: "审批管理", ImpliedReadPermission: domain.PermissionWorkflowRead},
	{Key: domain.PermissionCRLRead, Name: "查看 CRL", Description: "查看和下载证书撤销列表", Category: "证书撤销"},
	{Key: domain.PermissionCRLManage, Name: "管理 CRL", Description: "导出、下载和更新证书撤销列表", Category: "证书撤销", ImpliedReadPermission: domain.PermissionCRLRead},
	{Key: domain.PermissionAuditRead, Name: "查看审计", Description: "查看身份认证和 PKI 操作审计", Category: "审计日志"},
	{Key: domain.PermissionAuditManage, Name: "管理审计", Description: "导出审计日志", Category: "审计日志", ImpliedReadPermission: domain.PermissionAuditRead},
	{Key: domain.PermissionSettingsBaseRead, Name: "查看基础配置", Description: "查看平台、续期、CRL 和 OCSP 配置", Category: "系统配置"},
	{Key: domain.PermissionSettingsBaseManage, Name: "管理基础配置", Description: "维护平台、续期、CRL 和 OCSP 配置", Category: "系统配置", ImpliedReadPermission: domain.PermissionSettingsBaseRead},
	{Key: domain.PermissionSettingsUsersRead, Name: "查看用户配置", Description: "查看用户、用户群组和角色", Category: "系统配置"},
	{Key: domain.PermissionSettingsUsersManage, Name: "管理用户配置", Description: "维护用户、用户群组和角色", Category: "系统配置", ImpliedReadPermission: domain.PermissionSettingsUsersRead},
	{Key: domain.PermissionSettingsAuthRead, Name: "查看认证配置", Description: "查看本地账号和 LDAP 配置", Category: "系统配置"},
	{Key: domain.PermissionSettingsAuthManage, Name: "管理认证配置", Description: "维护并测试 LDAP 配置", Category: "系统配置", ImpliedReadPermission: domain.PermissionSettingsAuthRead},
	{Key: domain.PermissionSettingsNotifyRead, Name: "查看通知配置", Description: "查看找回密码邮件媒介", Category: "系统配置"},
	{Key: domain.PermissionSettingsNotifyManage, Name: "管理通知配置", Description: "维护并测试找回密码邮件媒介", Category: "系统配置", ImpliedReadPermission: domain.PermissionSettingsNotifyRead},
}

var (
	ErrRoleNotFound            = errors.New("role not found")
	ErrRoleIdentityImmutable   = errors.New("role identity immutable")
	ErrRoleMustBeDisabled      = errors.New("role must be disabled before deletion")
	ErrUserGroupMustBeDisabled = errors.New("user group must be disabled before deletion")
	ErrUserGroupNameImmutable  = errors.New("user group name immutable")
)

type RoleInput struct {
	Key         string
	Name        string
	Description string
	Permissions []string
}

type UserGroupInput struct {
	Name        string
	Description string
	Disabled    bool
	MemberIDs   []string
	RoleKeys    []string
}

func (s *Store) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text,key,name,description,permissions,builtin,disabled,created_at,updated_at
		FROM roles
		ORDER BY builtin DESC,CASE key WHEN 'admin' THEN 1 WHEN 'operator' THEN 2 WHEN 'viewer' THEN 3 ELSE 9 END,key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Role{}
	for rows.Next() {
		var item domain.Role
		if err := rows.Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Permissions = normalizePermissions(item.Permissions)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertCustomRole(ctx context.Context, id string, input RoleInput) (domain.Role, error) {
	input.Key = strings.TrimSpace(input.Key)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Permissions = normalizePermissions(input.Permissions)
	if input.Key == "" || input.Name == "" {
		return domain.Role{}, errors.New("角色标识和名称不能为空")
	}
	var item domain.Role
	if id == "" {
		id = uuid.NewString()
		err := s.Pool.QueryRow(ctx, `
			INSERT INTO roles(id,key,name,description,permissions,builtin)
			VALUES($1,$2,$3,$4,$5,FALSE)
			RETURNING id::text,key,name,description,permissions,builtin,disabled,created_at,updated_at
		`, id, input.Key, input.Name, input.Description, input.Permissions).Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt)
		return item, err
	}
	var currentKey, currentName string
	var builtin bool
	if err := s.Pool.QueryRow(ctx, "SELECT key,name,builtin FROM roles WHERE id=$1", id).Scan(&currentKey, &currentName, &builtin); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Role{}, ErrNotFound
		}
		return domain.Role{}, err
	}
	if builtin {
		return domain.Role{}, ErrNotFound
	}
	if currentKey != input.Key || currentName != input.Name {
		return domain.Role{}, ErrRoleIdentityImmutable
	}
	err := s.Pool.QueryRow(ctx, `
		UPDATE roles SET key=$2,name=$3,description=$4,permissions=$5,updated_at=now()
		WHERE id=$1 AND builtin=FALSE
		RETURNING id::text,key,name,description,permissions,builtin,disabled,created_at,updated_at
	`, id, input.Key, input.Name, input.Description, input.Permissions).Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Role{}, ErrNotFound
	}
	return item, err
}

func (s *Store) SetRoleDisabled(ctx context.Context, id string, disabled bool) (domain.Role, error) {
	var item domain.Role
	err := s.Pool.QueryRow(ctx, `
		UPDATE roles SET disabled=$2,updated_at=now()
		WHERE id=$1 AND builtin=FALSE
		RETURNING id::text,key,name,description,permissions,builtin,disabled,created_at,updated_at
	`, id, disabled).Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Role{}, ErrNotFound
	}
	return item, err
}

func (s *Store) DeleteCustomRole(ctx context.Context, id string) (string, error) {
	var name string
	var builtin bool
	var disabled bool
	if err := s.Pool.QueryRow(ctx, "SELECT name,builtin,disabled FROM roles WHERE id=$1", id).Scan(&name, &builtin, &disabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if builtin {
		return "", ErrNotFound
	}
	if !disabled {
		return "", ErrRoleMustBeDisabled
	}
	result, err := s.Pool.Exec(ctx, `
		DELETE FROM roles WHERE id=$1 AND builtin=FALSE
	`, id)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return name, nil
}

func (s *Store) SetUserRoles(ctx context.Context, userID string, roleKeys []string) error {
	roleKeys = normalizeRoleKeys(roleKeys)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM user_roles WHERE user_id=$1", userID); err != nil {
		return err
	}
	for _, key := range roleKeys {
		result, err := tx.Exec(ctx, `
			INSERT INTO user_roles(user_id,role_id)
			SELECT $1,id FROM roles WHERE key=$2 AND disabled=FALSE
		`, userID, key)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return ErrRoleNotFound
		}
	}
	_, err = tx.Exec(ctx, "UPDATE users SET role=$2,updated_at=now() WHERE id=$1", userID, roleKeys[0])
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AttachUserAccess(ctx context.Context, user domain.User) (domain.User, error) {
	direct, err := s.listDirectRoles(ctx, user.ID)
	if err != nil {
		return domain.User{}, err
	}
	effective, err := s.listEffectiveRoles(ctx, user.ID)
	if err != nil {
		return domain.User{}, err
	}
	user.DirectRoles = direct
	user.Roles = effective
	user.Permissions = uniquePermissions(effective)
	return user, nil
}

func (s *Store) ListUserGroups(ctx context.Context) ([]domain.UserGroup, error) {
	rows, err := s.Pool.Query(ctx, "SELECT id::text,name,description,disabled,created_at,updated_at FROM user_groups ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.UserGroup{}
	for rows.Next() {
		var item domain.UserGroup
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Disabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Members, err = s.groupMembers(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
		items[index].Roles, err = s.groupRoles(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Store) UserGroupDisabled(ctx context.Context, id string) (bool, error) {
	var disabled bool
	err := s.Pool.QueryRow(ctx, "SELECT disabled FROM user_groups WHERE id=$1", id).Scan(&disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return disabled, err
}

func (s *Store) UpsertUserGroup(ctx context.Context, id string, input UserGroupInput) (domain.UserGroup, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		return domain.UserGroup{}, errors.New("用户群组名称不能为空")
	}
	if id == "" {
		id = uuid.NewString()
		if _, err := s.Pool.Exec(ctx, "INSERT INTO user_groups(id,name,description,disabled) VALUES($1,$2,$3,$4)", id, input.Name, input.Description, input.Disabled); err != nil {
			return domain.UserGroup{}, err
		}
	} else {
		var currentName string
		if err := s.Pool.QueryRow(ctx, "SELECT name FROM user_groups WHERE id=$1", id).Scan(&currentName); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.UserGroup{}, ErrNotFound
			}
			return domain.UserGroup{}, err
		}
		if currentName != input.Name {
			return domain.UserGroup{}, ErrUserGroupNameImmutable
		}
		result, err := s.Pool.Exec(ctx, "UPDATE user_groups SET name=$2,description=$3,disabled=$4,updated_at=now() WHERE id=$1", id, input.Name, input.Description, input.Disabled)
		if err != nil {
			return domain.UserGroup{}, err
		}
		if result.RowsAffected() == 0 {
			return domain.UserGroup{}, ErrNotFound
		}
	}
	if err := s.replaceGroupMembers(ctx, id, input.MemberIDs); err != nil {
		return domain.UserGroup{}, err
	}
	if err := s.replaceGroupRoles(ctx, id, input.RoleKeys); err != nil {
		return domain.UserGroup{}, err
	}
	groups, err := s.ListUserGroups(ctx)
	if err != nil {
		return domain.UserGroup{}, err
	}
	for _, item := range groups {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.UserGroup{}, ErrNotFound
}

func (s *Store) DeleteUserGroup(ctx context.Context, id string) (string, error) {
	var name string
	var disabled bool
	if err := s.Pool.QueryRow(ctx, "SELECT name,disabled FROM user_groups WHERE id=$1", id).Scan(&name, &disabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if !disabled {
		return "", ErrUserGroupMustBeDisabled
	}
	result, err := s.Pool.Exec(ctx, "DELETE FROM user_groups WHERE id=$1", id)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return name, nil
}

func (s *Store) listDirectRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	return s.queryRoles(ctx, `
		SELECT r.id::text,r.key,r.name,r.description,r.permissions,r.builtin,r.disabled,r.created_at,r.updated_at
		FROM roles r JOIN user_roles ur ON ur.role_id=r.id
		WHERE ur.user_id=$1 ORDER BY CASE r.key WHEN 'admin' THEN 1 WHEN 'operator' THEN 2 WHEN 'viewer' THEN 3 ELSE 9 END,r.key
	`, userID)
}

func (s *Store) listEffectiveRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	return s.queryRoles(ctx, `
		SELECT id,key,name,description,permissions,builtin,disabled,created_at,updated_at
		FROM (
			SELECT DISTINCT r.id::text,r.key,r.name,r.description,r.permissions,r.builtin,r.disabled,r.created_at,r.updated_at,
				CASE r.key WHEN 'admin' THEN 1 WHEN 'operator' THEN 2 WHEN 'viewer' THEN 3 ELSE 9 END AS sort_order
			FROM roles r WHERE r.disabled=FALSE AND r.id IN (
				SELECT role_id FROM user_roles WHERE user_id=$1
				UNION
				SELECT ugr.role_id FROM user_group_roles ugr
				JOIN user_group_members ugm ON ugm.group_id=ugr.group_id
				JOIN user_groups g ON g.id=ugm.group_id AND g.disabled=FALSE
				WHERE ugm.user_id=$1
			)
		) effective_roles
		ORDER BY sort_order,key
	`, userID)
}

func (s *Store) queryRoles(ctx context.Context, query string, arg string) ([]domain.Role, error) {
	rows, err := s.Pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Role{}
	for rows.Next() {
		var item domain.Role
		if err := rows.Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) roleByKey(ctx context.Context, key string) (domain.Role, error) {
	var item domain.Role
	err := s.Pool.QueryRow(ctx, "SELECT id::text,key,name,description,permissions,builtin,disabled,created_at,updated_at FROM roles WHERE key=$1 AND disabled=FALSE", key).Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Permissions, &item.Builtin, &item.Disabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) replaceGroupMembers(ctx context.Context, groupID string, memberIDs []string) error {
	if _, err := s.Pool.Exec(ctx, "DELETE FROM user_group_members WHERE group_id=$1", groupID); err != nil {
		return err
	}
	for _, userID := range normalizeIDs(memberIDs) {
		if _, err := s.Pool.Exec(ctx, "INSERT INTO user_group_members(group_id,user_id) VALUES($1,$2)", groupID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) replaceGroupRoles(ctx context.Context, groupID string, roleKeys []string) error {
	if _, err := s.Pool.Exec(ctx, "DELETE FROM user_group_roles WHERE group_id=$1", groupID); err != nil {
		return err
	}
	for _, key := range normalizeRoleKeys(roleKeys) {
		result, err := s.Pool.Exec(ctx, "INSERT INTO user_group_roles(group_id,role_id) SELECT $1,id FROM roles WHERE key=$2 AND disabled=FALSE", groupID, key)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return ErrRoleNotFound
		}
	}
	return nil
}

func (s *Store) groupMembers(ctx context.Context, groupID string) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT u.id::text,u.username,u.display_name,u.email,u.password_hash,u.role,u.source,u.disabled,u.last_login_at,u.created_at,u.updated_at
		FROM users u JOIN user_group_members m ON m.user_id=u.id WHERE m.group_id=$1 ORDER BY u.username
	`, groupID)
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
		items = append(items, user)
	}
	return items, rows.Err()
}

func (s *Store) groupRoles(ctx context.Context, groupID string) ([]domain.Role, error) {
	return s.queryRoles(ctx, `
		SELECT r.id::text,r.key,r.name,r.description,r.permissions,r.builtin,r.disabled,r.created_at,r.updated_at
		FROM roles r JOIN user_group_roles gr ON gr.role_id=r.id WHERE gr.group_id=$1 ORDER BY r.key
	`, groupID)
}

func normalizePermissions(values []string) []string {
	valid := map[string]domain.Permission{}
	for _, permission := range BuiltinPermissions {
		valid[permission.Key] = permission
	}
	seen := map[string]struct{}{}
	queue := append([]string(nil), values...)
	for len(queue) > 0 {
		value := queue[0]
		queue = queue[1:]
		value = strings.TrimSpace(value)
		permission, ok := valid[value]
		if !ok || value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		if permission.ImpliedReadPermission != "" {
			queue = append(queue, permission.ImpliedReadPermission)
		}
		queue = append(queue, permission.ImpliedPermissions...)
	}
	items := make([]string, 0, len(seen))
	for value := range seen {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func normalizeRoleKeys(values []string) []string {
	items := normalizeRoleKeysAllowEmpty(values)
	if len(items) == 0 {
		return []string{"viewer"}
	}
	return items
}

func normalizeRoleKeysAllowEmpty(values []string) []string {
	seen := map[string]struct{}{}
	items := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func normalizeIDs(values []string) []string {
	seen := map[string]struct{}{}
	items := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func uniquePermissions(roles []domain.Role) []string {
	permissions := []string{}
	for _, role := range roles {
		permissions = append(permissions, role.Permissions...)
	}
	return normalizePermissions(permissions)
}
