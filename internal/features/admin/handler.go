package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

// Handler menangani /api/admin/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler admin.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
}

// ListUsers: GET /api/admin/users
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListUsers()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar pengguna berhasil diambil", users)
}

// ListPosts: GET /api/admin/posts
func (h *Handler) ListPosts(c *gin.Context) {
	posts, err := h.svc.ListPosts()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Arsip artikel berhasil diambil", posts)
}

// ListUserPosts: GET /api/admin/users/:id/posts
func (h *Handler) ListUserPosts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	res, err := h.svc.ListUserPosts(uint(id))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel pengguna berhasil diambil", res)
}
