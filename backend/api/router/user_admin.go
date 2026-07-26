package router

import (
	"errors"
	"net/http"
	"strings"

	"certflow/backend/internal/repository"
)

func (r *Router) deleteUser(w http.ResponseWriter, q *http.Request) {
	id := q.PathValue("id")
	if id == current(q).ID {
		write(w, http.StatusBadRequest, map[string]string{"message": "不能删除当前账号"})
		return
	}
	user, err := r.store.FindUserByID(q.Context(), id)
	if err != nil {
		writeDeleteUserError(w, err)
		return
	}
	if err := r.store.DeleteUser(q.Context(), id); err != nil {
		writeDeleteUserError(w, err)
		return
	}
	r.store.Audit(q.Context(), current(q), "删除用户", user.Username, "settings", "success", clientIP(q), "用户 "+user.Username+" 已删除")
	w.WriteHeader(http.StatusNoContent)
}

func writeDeleteUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		write(w, http.StatusNotFound, map[string]string{"message": "用户不存在"})
	case errors.Is(err, repository.ErrDefaultAdminProtected):
		write(w, http.StatusBadRequest, map[string]string{"message": "默认管理员不能删除"})
	case errors.Is(err, repository.ErrUserMustBeDisabled):
		write(w, http.StatusBadRequest, map[string]string{"message": "请先禁用用户再删除"})
	default:
		write(w, http.StatusInternalServerError, map[string]string{"message": "删除用户失败"})
	}
}

func (r *Router) changePassword(w http.ResponseWriter, q *http.Request) {
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	if len(body.NewPassword) < 6 || strings.TrimSpace(body.CurrentPassword) == "" {
		write(w, http.StatusBadRequest, map[string]string{"message": "当前密码和至少 6 位新密码必填"})
		return
	}
	user := current(q)
	if err := r.store.ChangePassword(q.Context(), user.ID, body.CurrentPassword, body.NewPassword); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	r.store.Audit(q.Context(), user, "修改密码", user.Username, "auth", "success", clientIP(q), "用户 "+user.Username+" 修改密码成功")
	w.WriteHeader(http.StatusNoContent)
}
