package router

import (
	"net/http"

	pkisvc "certflow/backend/internal/service/pki"
)

func (r *Router) importCA(w http.ResponseWriter, q *http.Request) {
	var body pkisvc.ImportCAInput
	if err := decode(q, &body); err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "请求格式不正确"})
		return
	}
	ca, err := r.pki.ImportCA(q.Context(), body, current(q), clientIP(q))
	if err != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	write(w, http.StatusCreated, ca)
}
