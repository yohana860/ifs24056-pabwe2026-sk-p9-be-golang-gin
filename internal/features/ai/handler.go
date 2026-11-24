package ai

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrSearchNotConfigured):
		return http.StatusServiceUnavailable, "Layanan OpenAI atau Search Engine belum dikonfigurasi" // <-- DIUBAH
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "Layanan OpenAI atau Search Engine melewati batas waktu" // <-- DIUBAH
	case errors.Is(err, ErrSearchFailed), errors.Is(err, ErrSearchEmpty):
		return http.StatusBadGateway, "Penelusuran Search Engine gagal"
	case errors.Is(err, ErrUpstream), errors.Is(err, ErrInvalidResponse),
		errors.Is(err, ErrEmptyResponse), errors.Is(err, ErrInvalidAI):
		return http.StatusBadGateway, "Layanan AI gagal menghasilkan artikel"
	}
	return http.StatusInternalServerError, "Terjadi kesalahan pada server"
}

func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err)
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