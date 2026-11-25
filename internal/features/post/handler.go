package post

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func uid(c *gin.Context) uint {
	user := middleware.CurrentUser(c)
	if user == nil {
		return 0
	}
	return user.UserID
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidCover), errors.Is(err, ErrInvalidCategory):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func (h *Handler) Categories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Daftar kategori berhasil diambil", h.svc.Categories())
}

func (h *Handler) ListPublic(c *gin.Context) {
	posts, err := h.svc.ListPublic(Filter{Q: c.Query("q"), Category: c.Query("category")})
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel berhasil diambil", posts)
}

func (h *Handler) GetPublic(c *gin.Context) {
	p, err := h.svc.GetPublic(c.Param("slug"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

func (h *Handler) ListMine(c *gin.Context) {
	posts, err := h.svc.ListMine(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel Anda berhasil diambil", posts)
}

func (h *Handler) Stats(c *gin.Context) {
	st, err := h.svc.Stats(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik artikel berhasil diambil", st)
}

func (h *Handler) GetMine(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetMine(uid(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

func (h *Handler) Create(c *gin.Context) {
	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	p, err := h.svc.Create(uid(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", p)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdatePostRequest // <-- INI YANG BENAR
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	p, err := h.svc.Update(uid(c), id, req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", p)
}

func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.TogglePublish(uid(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Status publikasi berhasil diperbarui", p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(uid(c), id); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil dihapus", nil)
}