package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

// Handler menangani endpoint /api/auth/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler autentikasi.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrConflict):
		response.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		response.Fail(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// Register godoc: POST /api/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	user, err := h.svc.Register(req)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Registrasi berhasil", user)
}

// Login godoc: POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	res, err := h.svc.Login(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Login berhasil", res)
}

// Logout godoc: POST /api/auth/logout (JWT stateless; klien menghapus token).
func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// Profile godoc: GET /api/auth/profile
func (h *Handler) Profile(c *gin.Context) {
	user, err := h.svc.Profile(middleware.CurrentUser(c).UserID)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diambil", user)
}

// UpdateProfile godoc: PUT /api/auth/profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	user, err := h.svc.UpdateProfile(middleware.CurrentUser(c).UserID, req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", user)
}
