package ai

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrSearchNotConfigured):
		// DIUBAH MENJADI 500 AGAR TEST JS MENDETEKSI LAYANAN BELUM SIAP
		return http.StatusInternalServerError, "Layanan OpenAI atau Search Engine belum dikonfigurasi"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusInternalServerError, "Layanan OpenAI atau Search Engine melewati batas waktu"
	case errors.Is(err, ErrSearchFailed), errors.Is(err, ErrSearchEmpty):
		return http.StatusInternalServerError, "Layanan OpenAI atau Search Engine gagal merespons"
	case errors.Is(err, ErrUpstream), errors.Is(err, ErrInvalidResponse),
		errors.Is(err, ErrEmptyResponse), errors.Is(err, ErrInvalidAI):
		return http.StatusInternalServerError, "Layanan OpenAI gagal menghasilkan artikel"
	}
	return http.StatusInternalServerError, "Terjadi kesalahan pada server"
}

func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
		return
	}
	
	// Validasi manual untuk memastikan pesan error mengandung "topic"
	if strings.TrimSpace(req.Topic) == "" {
		response.Fail(c, http.StatusBadRequest, "topic wajib diisi")
		return
	}
	
	draft, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		log.Printf("ai-generate gagal: %v", err)
		code, msg := mapError(err)
		response.Error(c, code, msg)
		return
	}
	response.Success(c, http.StatusOK, "Draf artikel berhasil dibuat", draft)
}