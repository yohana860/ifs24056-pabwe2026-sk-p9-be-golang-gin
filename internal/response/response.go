// Package response menstandarkan format balasan JSON {status, message, data?}.
package response

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Body adalah struktur balasan standar.
type Body struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success mengirim balasan sukses (HTTP 2xx).
func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Body{Status: "success", Message: message, Data: data})
}

// Fail mengirim kesalahan klien (HTTP 4xx).
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "fail", Message: message})
}

// Error mengirim kesalahan server (HTTP 5xx).
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "error", Message: message})
}

// AbortFail seperti Fail namun menghentikan rantai middleware.
func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "fail", Message: message})
}

// AbortError seperti Error namun menghentikan rantai middleware.
func AbortError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "error", Message: message})
}

// Validation mengubah kesalahan binding menjadi balasan 400 yang tidak membocorkan detail internal.
func Validation(c *gin.Context, err error) {
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		Fail(c, http.StatusBadRequest, "Format permintaan tidak valid")
		return
	}
	fields := map[string]string{}
	for _, fe := range verrs {
		fields[toSnake(fe.Field())] = ruleMessage(fe)
	}
	c.JSON(http.StatusBadRequest, Body{Status: "fail", Message: "Validasi gagal", Data: gin.H{"errors": fields}})
}

func ruleMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "min":
		return "terlalu pendek (minimal " + fe.Param() + ")"
	case "max":
		return "terlalu panjang (maksimal " + fe.Param() + ")"
	case "email":
		return "format email tidak valid"
	case "oneof":
		return "harus salah satu dari: " + fe.Param()
	}
	return "tidak valid"
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
