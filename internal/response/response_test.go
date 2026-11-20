package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func do(h gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/", h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) Body {
	var b Body
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b
}

func TestStatusHelpers(t *testing.T) {
	w := do(func(c *gin.Context) { Success(c, 201, "ok", gin.H{"a": 1}) }, "")
	assert.Equal(t, 201, w.Code)
	b := decode(t, w)
	assert.Equal(t, "success", b.Status)
	assert.NotNil(t, b.Data)

	w = do(func(c *gin.Context) { Fail(c, 400, "bad") }, "")
	assert.Equal(t, "fail", decode(t, w).Status)

	w = do(func(c *gin.Context) { Error(c, 500, "boom") }, "")
	assert.Equal(t, "error", decode(t, w).Status)
	assert.Nil(t, decode(t, w).Data)

	w = do(func(c *gin.Context) { AbortFail(c, 401, "no") }, "")
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "fail", decode(t, w).Status)

	w = do(func(c *gin.Context) { AbortError(c, 503, "down") }, "")
	assert.Equal(t, 503, w.Code)
	assert.Equal(t, "error", decode(t, w).Status)
}

type sample struct {
	UserName string `json:"user_name" binding:"required"`
	Short    string `json:"short" binding:"min=3"`
	Long     string `json:"long" binding:"max=2"`
	Mail     string `json:"mail" binding:"email"`
	Kind     string `json:"kind" binding:"oneof=a b"`
	Count    int    `json:"count" binding:"gte=5"`
}

func TestValidation(t *testing.T) {
	h := func(c *gin.Context) {
		var s sample
		if err := c.ShouldBindJSON(&s); err != nil {
			Validation(c, err)
			return
		}
		Success(c, 200, "ok", nil)
	}
	w := do(h, `{"short":"a","long":"abcd","mail":"x","kind":"z","count":1}`)
	assert.Equal(t, 400, w.Code)
	var raw struct {
		Data struct {
			Errors map[string]string `json:"errors"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	e := raw.Data.Errors
	assert.Equal(t, "wajib diisi", e["user_name"])
	assert.Contains(t, e["short"], "terlalu pendek")
	assert.Contains(t, e["long"], "terlalu panjang")
	assert.Contains(t, e["mail"], "email")
	assert.Contains(t, e["kind"], "salah satu")
	assert.Equal(t, "tidak valid", e["count"])

	w = do(h, `{not json`)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, "Format permintaan tidak valid", decode(t, w).Message)

	w = do(h, `{"user_name":"x","short":"abc","long":"a","mail":"a@b.co","kind":"a","count":9}`)
	assert.Equal(t, 200, w.Code)
}
