package router

import (
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"certflow/backend/internal/repository"
	authsvc "certflow/backend/internal/service/auth"
)

var (
	errInvalidConfigurationRequest = errors.New("请求格式不正确")
	managedRoleKeyPattern          = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,40}$`)
)

func (r *Router) listPermissions(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"items": repository.BuiltinPermissions, "total": len(repository.BuiltinPermissions)})
}

func (r *Router) listRoles(w http.ResponseWriter, q *http.Request) {
	items, err := r.store.ListRoles(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取用户角色失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func decodeRole(q *http.Request) (repository.RoleInput, error) {
	var body struct {
		Key         string   `json:"key"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := decode(q, &body); err != nil {
		return repository.RoleInput{}, errInvalidConfigurationRequest
	}
	body.Key = strings.TrimSpace(body.Key)
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	body.Permissions = normalizeRequestStrings(body.Permissions)
	if !managedRoleKeyPattern.MatchString(body.Key) {
		return repository.RoleInput{}, errors.New("角色标识需为小写字母、数字、点、下划线或连字符")
	}
	if body.Name == "" {
		return repository.RoleInput{}, errors.New("角色名称不能为空")
	}
	return repository.RoleInput{Key: body.Key, Name: body.Name, Description: body.Description, Permissions: body.Permissions}, nil
}

func (r *Router) createRole(w http.ResponseWriter, q *http.Request) {
	input, err := decodeRole(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	item, err := r.store.UpsertCustomRole(q.Context(), "", input)
	if err != nil {
		writeRoleSaveError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "创建用户角色", item.Name, "settings", "success", clientIP(q), "用户角色 "+item.Name+" 已创建")
	write(w, http.StatusCreated, item)
}

func (r *Router) updateRole(w http.ResponseWriter, q *http.Request) {
	input, err := decodeRole(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	item, err := r.store.UpsertCustomRole(q.Context(), q.PathValue("id"), input)
	if err != nil {
		writeRoleSaveError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "更新用户角色", item.Name, "settings", "success", clientIP(q), "用户角色 "+item.Name+" 已更新")
	write(w, http.StatusOK, item)
}

func writeRoleSaveError(w http.ResponseWriter, err error) {
	if repository.IsUniqueViolation(err) {
		write(w, http.StatusConflict, map[string]string{"message": "角色标识已存在"})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		write(w, http.StatusNotFound, map[string]string{"message": "内置角色不可修改或角色不存在"})
		return
	}
	if errors.Is(err, repository.ErrRoleIdentityImmutable) {
		write(w, http.StatusBadRequest, map[string]string{"message": "用户角色创建后标识和名称不可修改"})
		return
	}
	write(w, http.StatusInternalServerError, map[string]string{"message": "保存用户角色失败"})
}

func (r *Router) setRoleDisabled(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	item, err := r.store.SetRoleDisabled(q.Context(), q.PathValue("id"), body.Disabled)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			write(w, http.StatusNotFound, map[string]string{"message": "内置角色不可禁用或角色不存在"})
			return
		}
		write(w, http.StatusInternalServerError, map[string]string{"message": "更新用户角色状态失败"})
		return
	}
	roleStatus := "已启用"
	if item.Disabled {
		roleStatus = "已禁用"
	}
	roleAction := "启用用户角色"
	if item.Disabled {
		roleAction = "禁用用户角色"
	}
	r.store.Audit(q.Context(), current(q), roleAction, item.Name, "settings", "success", clientIP(q), "用户角色 "+item.Name+" 状态更新为 "+roleStatus)
	write(w, http.StatusOK, item)
}

func (r *Router) deleteRole(w http.ResponseWriter, q *http.Request) {
	name, err := r.store.DeleteCustomRole(q.Context(), q.PathValue("id"))
	if err != nil {
		writeRoleDeleteError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "删除用户角色", name, "settings", "success", clientIP(q), "用户角色 "+name+" 已删除")
	w.WriteHeader(http.StatusNoContent)
}

func writeRoleDeleteError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrRoleMustBeDisabled) {
		write(w, http.StatusBadRequest, map[string]string{"message": "请先禁用用户角色再删除"})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		write(w, http.StatusNotFound, map[string]string{"message": "内置角色不可删除或角色不存在"})
		return
	}
	write(w, http.StatusInternalServerError, map[string]string{"message": "删除用户角色失败"})
}

func (r *Router) listUserGroups(w http.ResponseWriter, q *http.Request) {
	items, err := r.store.ListUserGroups(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取用户群组失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func decodeUserGroup(q *http.Request) (repository.UserGroupInput, error) {
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Disabled    bool     `json:"disabled"`
		MemberIDs   []string `json:"memberIds"`
		RoleKeys    []string `json:"roleKeys"`
	}
	if err := decode(q, &body); err != nil {
		return repository.UserGroupInput{}, errInvalidConfigurationRequest
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	body.MemberIDs = normalizeRequestStrings(body.MemberIDs)
	body.RoleKeys = normalizeRequestStrings(body.RoleKeys)
	if body.Name == "" {
		return repository.UserGroupInput{}, errors.New("用户群组名称不能为空")
	}
	return repository.UserGroupInput{Name: body.Name, Description: body.Description, Disabled: body.Disabled, MemberIDs: body.MemberIDs, RoleKeys: body.RoleKeys}, nil
}

func (r *Router) createUserGroup(w http.ResponseWriter, q *http.Request) {
	input, err := decodeUserGroup(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	item, err := r.store.UpsertUserGroup(q.Context(), "", input)
	if err != nil {
		writeUserGroupSaveError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "创建用户群组", item.Name, "settings", "success", clientIP(q), "用户群组 "+item.Name+" 已创建")
	write(w, http.StatusCreated, item)
}

func (r *Router) updateUserGroup(w http.ResponseWriter, q *http.Request) {
	input, err := decodeUserGroup(q)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	wasDisabled, err := r.store.UserGroupDisabled(q.Context(), q.PathValue("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			write(w, http.StatusNotFound, map[string]string{"message": "用户群组不存在"})
			return
		}
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取用户群组状态失败"})
		return
	}
	item, err := r.store.UpsertUserGroup(q.Context(), q.PathValue("id"), input)
	if err != nil {
		writeUserGroupSaveError(w, err)
		return
	}
	groupAction := "更新用户群组"
	groupDetail := "用户群组 " + item.Name + " 已更新"
	if wasDisabled != item.Disabled {
		groupStatus := "已启用"
		groupAction = "启用用户群组"
		if item.Disabled {
			groupStatus = "已禁用"
			groupAction = "禁用用户群组"
		}
		groupDetail = "用户群组 " + item.Name + " 状态更新为 " + groupStatus
	}
	r.store.Audit(q.Context(), current(q), groupAction, item.Name, "settings", "success", clientIP(q), groupDetail)
	write(w, http.StatusOK, item)
}

func writeUserGroupSaveError(w http.ResponseWriter, err error) {
	if repository.IsUniqueViolation(err) {
		write(w, http.StatusConflict, map[string]string{"message": "用户群组名称已存在"})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		write(w, http.StatusNotFound, map[string]string{"message": "用户群组不存在"})
		return
	}
	if errors.Is(err, repository.ErrUserGroupNameImmutable) {
		write(w, http.StatusBadRequest, map[string]string{"message": "用户群组创建后名称不可修改"})
		return
	}
	if errors.Is(err, repository.ErrRoleNotFound) {
		write(w, http.StatusBadRequest, map[string]string{"message": "选择的角色不存在"})
		return
	}
	write(w, http.StatusInternalServerError, map[string]string{"message": "保存用户群组失败"})
}

func (r *Router) deleteUserGroup(w http.ResponseWriter, q *http.Request) {
	name, err := r.store.DeleteUserGroup(q.Context(), q.PathValue("id"))
	if err != nil {
		if errors.Is(err, repository.ErrUserGroupMustBeDisabled) {
			write(w, http.StatusBadRequest, map[string]string{"message": "请先禁用用户群组再删除"})
			return
		}
		if !errors.Is(err, repository.ErrNotFound) {
			write(w, http.StatusInternalServerError, map[string]string{"message": "删除用户群组失败"})
			return
		}
		write(w, http.StatusNotFound, map[string]string{"message": "用户群组不存在"})
		return
	}
	r.store.Audit(q.Context(), current(q), "删除用户群组", name, "settings", "success", clientIP(q), "用户群组 "+name+" 已删除")
	w.WriteHeader(http.StatusNoContent)
}

func decodeManagedUser(q *http.Request, create bool) (repository.ManagedUserInput, error) {
	var body struct {
		Username    string   `json:"username"`
		DisplayName string   `json:"displayName"`
		Email       string   `json:"email"`
		Password    string   `json:"password"`
		RoleKeys    []string `json:"roleKeys"`
		Disabled    bool     `json:"disabled"`
	}
	if err := decode(q, &body); err != nil {
		return repository.ManagedUserInput{}, errInvalidConfigurationRequest
	}
	body.Username = strings.TrimSpace(body.Username)
	body.DisplayName = strings.TrimSpace(body.DisplayName)
	body.Email = strings.TrimSpace(body.Email)
	body.RoleKeys = normalizeRequestStrings(body.RoleKeys)
	if body.Username == "" {
		return repository.ManagedUserInput{}, errors.New("用户名不能为空")
	}
	address, err := mail.ParseAddress(body.Email)
	if body.Email == "" || err != nil || address.Address != body.Email {
		return repository.ManagedUserInput{}, errors.New("请输入有效的邮箱地址")
	}
	if body.DisplayName == "" {
		body.DisplayName = body.Username
	}
	if create && len(body.Password) < 6 {
		return repository.ManagedUserInput{}, errors.New("密码至少 6 个字符")
	}
	if !create && body.Password != "" && len(body.Password) < 6 {
		return repository.ManagedUserInput{}, errors.New("密码至少 6 个字符")
	}
	if len(body.RoleKeys) == 0 {
		body.RoleKeys = []string{"viewer"}
	}
	return repository.ManagedUserInput{Username: body.Username, DisplayName: body.DisplayName, Email: body.Email, Password: body.Password, RoleKeys: body.RoleKeys, Disabled: body.Disabled}, nil
}

func (r *Router) createManagedUser(w http.ResponseWriter, q *http.Request) {
	input, err := decodeManagedUser(q, true)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	item, err := r.store.CreateManagedUser(q.Context(), input)
	if err != nil {
		writeManagedUserSaveError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "创建用户", item.Username, "settings", "success", clientIP(q), "本地用户 "+item.Username+" 已创建")
	write(w, http.StatusCreated, item)
}

func (r *Router) updateManagedUser(w http.ResponseWriter, q *http.Request) {
	id := q.PathValue("id")
	input, err := decodeManagedUser(q, false)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	if id == current(q).ID && input.Disabled {
		write(w, http.StatusBadRequest, map[string]string{"message": "不能禁用当前账号"})
		return
	}
	item, err := r.store.UpdateManagedUser(q.Context(), id, input)
	if err != nil {
		writeManagedUserSaveError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "更新用户", item.Username, "settings", "success", clientIP(q), "用户 "+item.Username+" 已更新")
	write(w, http.StatusOK, item)
}

func writeManagedUserSaveError(w http.ResponseWriter, err error) {
	if repository.IsUniqueViolation(err) {
		write(w, http.StatusConflict, map[string]string{"message": "用户名已存在"})
		return
	}
	if errors.Is(err, repository.ErrRoleNotFound) {
		write(w, http.StatusBadRequest, map[string]string{"message": "选择的角色不存在"})
		return
	}
	if errors.Is(err, repository.ErrDefaultAdminProtected) {
		write(w, http.StatusBadRequest, map[string]string{"message": "默认管理员账号不能改名或禁用"})
		return
	}
	if errors.Is(err, repository.ErrManagedUserIdentityImmutable) {
		write(w, http.StatusBadRequest, map[string]string{"message": "用户创建后用户名不可修改"})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		write(w, http.StatusNotFound, map[string]string{"message": "用户不存在"})
		return
	}
	write(w, http.StatusInternalServerError, map[string]string{"message": "保存用户失败"})
}

func (r *Router) setSettingsUserStatus(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	id := q.PathValue("id")
	if id == current(q).ID && body.Disabled {
		write(w, http.StatusBadRequest, map[string]string{"message": "不能禁用当前账号"})
		return
	}
	user, err := r.store.FindUserByID(q.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			write(w, http.StatusNotFound, map[string]string{"message": "用户不存在"})
			return
		}
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取用户失败"})
		return
	}
	if user.Username == "admin" && body.Disabled {
		write(w, http.StatusBadRequest, map[string]string{"message": "默认管理员不能禁用"})
		return
	}
	if err := r.store.SetUserDisabled(q.Context(), id, body.Disabled); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			write(w, http.StatusNotFound, map[string]string{"message": "用户不存在"})
			return
		}
		write(w, http.StatusInternalServerError, map[string]string{"message": "更新用户状态失败"})
		return
	}
	user.Disabled = body.Disabled
	userStatus := "已启用"
	if user.Disabled {
		userStatus = "已禁用"
	}
	userAction := "启用用户"
	if user.Disabled {
		userAction = "禁用用户"
	}
	r.store.Audit(q.Context(), current(q), userAction, user.Username, "settings", "success", clientIP(q), "用户 "+user.Username+" 状态更新为 "+userStatus)
	write(w, http.StatusOK, user)
}

func normalizeRequestStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func (r *Router) getAuthProvider(w http.ResponseWriter, q *http.Request) {
	item, err := r.auth.LDAPSetting(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取认证配置失败"})
		return
	}
	write(w, http.StatusOK, item)
}

func (r *Router) saveAuthProvider(w http.ResponseWriter, q *http.Request) {
	var body struct {
		Name        string         `json:"name"`
		Enabled     bool           `json:"enabled"`
		ClearConfig bool           `json:"clearConfig"`
		Config      map[string]any `json:"config"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	var item repository.AuthProviderSetting
	var err error
	if body.ClearConfig {
		item, err = r.auth.ClearLDAPSetting(q.Context())
	} else {
		item, err = r.auth.SaveLDAPSetting(q.Context(), body.Name, body.Enabled, body.Config)
	}
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), current(q), "更新认证配置", "AD/LDAP", "settings", "success", clientIP(q), "AD/LDAP 认证配置已更新")
	write(w, http.StatusOK, item)
}

func (r *Router) testAuthProvider(w http.ResponseWriter, q *http.Request) {
	matched, err := r.auth.TestLDAPSetting(q.Context())
	if err != nil {
		write(w, http.StatusServiceUnavailable, map[string]string{"message": authsvc.LDAPUserMessage(err)})
		return
	}
	r.store.Audit(q.Context(), current(q), "测试认证配置", "AD/LDAP", "settings", "success", clientIP(q), "AD/LDAP 认证配置测试成功，匹配用户数："+strconv.Itoa(matched))
	write(w, http.StatusOK, map[string]any{"status": "ok", "matchedUsers": matched})
}

func (r *Router) getEmailSetting(w http.ResponseWriter, q *http.Request) {
	item, err := r.notify.Setting(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取邮件配置失败"})
		return
	}
	write(w, http.StatusOK, item)
}

func (r *Router) saveEmailSetting(w http.ResponseWriter, q *http.Request) {
	var body struct {
		PasswordResetEnabled bool           `json:"passwordResetEnabled"`
		ClearConfig          bool           `json:"clearConfig"`
		Config               map[string]any `json:"config"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	item, err := r.notify.SaveSetting(q.Context(), body.PasswordResetEnabled, body.ClearConfig, body.Config)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), current(q), "更新邮件配置", "找回密码邮件", "settings", "success", clientIP(q), "找回密码邮件配置已更新")
	write(w, http.StatusOK, item)
}

func (r *Router) testEmailSetting(w http.ResponseWriter, q *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if err := decode(q, &body); err != nil || strings.TrimSpace(body.To) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "请输入测试邮箱"})
		return
	}
	if err := r.notify.Test(q.Context(), body.To); err != nil {
		write(w, http.StatusServiceUnavailable, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), current(q), "测试邮件配置", "找回密码邮件", "settings", "success", clientIP(q), "找回密码邮件测试成功")
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) listNotificationChannels(w http.ResponseWriter, q *http.Request) {
	items, err := r.notify.Settings(q.Context())
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取通知配置失败"})
		return
	}
	write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (r *Router) saveNotificationChannel(w http.ResponseWriter, q *http.Request) {
	var body struct {
		PasswordResetEnabled bool           `json:"passwordResetEnabled"`
		ApprovalEnabled      bool           `json:"approvalEnabled"`
		ClearConfig          bool           `json:"clearConfig"`
		Config               map[string]any `json:"config"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	item, err := r.notify.SaveChannel(q.Context(), q.PathValue("id"), body.PasswordResetEnabled, body.ApprovalEnabled, body.ClearConfig, body.Config)
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), current(q), "更新通知配置", item.Name, "settings", "success", clientIP(q), "通知媒介配置已更新")
	write(w, http.StatusOK, item)
}

func (r *Router) testNotificationChannel(w http.ResponseWriter, q *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	if err := r.notify.TestChannel(q.Context(), q.PathValue("id"), body.To); err != nil {
		write(w, http.StatusServiceUnavailable, map[string]string{"message": err.Error()})
		return
	}
	channel, err := r.store.NotificationChannelSetting(q.Context(), q.PathValue("id"))
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"message": "读取通知配置失败"})
		return
	}
	r.store.Audit(q.Context(), current(q), "测试通知配置", channel.Name, "settings", "success", clientIP(q), "通知媒介测试发送成功")
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}
