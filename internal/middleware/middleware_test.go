package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "unit-test-secret-unit-test-secret-1234"

func newEngine(mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw...)
	r.Any("/x", func(c *gin.Context) {
		u := CurrentUser(c)
		name := ""
		if u != nil {
			name = u.Username
		}
		c.String(http.StatusOK, name)
	})
	return r
}

func call(r *gin.Engine, method string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestAuth(t *testing.T) {
	r := newEngine(Auth(secret))
	good, err := GenerateToken(secret, 7, "yobez", "member", time.Hour)
	require.NoError(t, err)
	expired, _ := GenerateToken(secret, 7, "yobez", "member", -time.Hour)
	wrongKey, _ := GenerateToken("other-secret-other-secret-other-secret", 7, "yobez", "member", time.Hour)
	noneAlg, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: 1}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	assert.Equal(t, 401, call(r, "GET", nil).Code)
	assert.Equal(t, 401, call(r, "GET", map[string]string{"Authorization": "Basic abc"}).Code)
	assert.Equal(t, 401, call(r, "GET", map[string]string{"Authorization": "Bearer garbage"}).Code)
	assert.Equal(t, 401, call(r, "GET", map[string]string{"Authorization": "Bearer " + expired}).Code)
	assert.Equal(t, 401, call(r, "GET", map[string]string{"Authorization": "Bearer " + wrongKey}).Code)
	assert.Equal(t, 401, call(r, "GET", map[string]string{"Authorization": "Bearer " + noneAlg}).Code)

	w := call(r, "GET", map[string]string{"Authorization": "bearer " + good})
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "yobez", w.Body.String())
}

func TestCurrentUserWithoutAuth(t *testing.T) {
	w := call(newEngine(), "GET", nil)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestRequireRole(t *testing.T) {
	admin, _ := GenerateToken(secret, 1, "boss", "admin", time.Hour)
	member, _ := GenerateToken(secret, 2, "m", "member", time.Hour)
	r := newEngine(Auth(secret), RequireRole("admin"))
	assert.Equal(t, 200, call(r, "GET", map[string]string{"Authorization": "Bearer " + admin}).Code)
	assert.Equal(t, 403, call(r, "GET", map[string]string{"Authorization": "Bearer " + member}).Code)
	// tanpa Auth: claims nil -> ditolak
	assert.Equal(t, 403, call(newEngine(RequireRole("admin")), "GET", nil).Code)
}

func TestCORS(t *testing.T) {
	star := newEngine(CORS([]string{"*"}))
	w := call(star, "GET", map[string]string{"Origin": "http://x.test"})
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	w = call(star, "OPTIONS", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	strict := newEngine(CORS([]string{"http://ok.test"}))
	w = call(strict, "GET", map[string]string{"Origin": "http://ok.test"})
	assert.Equal(t, "http://ok.test", w.Header().Get("Access-Control-Allow-Origin"))
	w = call(strict, "GET", map[string]string{"Origin": "http://evil.test"})
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	w = call(strict, "GET", nil)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurityHeaders(t *testing.T) {
	w := call(newEngine(SecurityHeaders()), "GET", nil)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Timeout(time.Minute))
	r.GET("/x", func(c *gin.Context) {
		_, ok := c.Request.Context().Deadline()
		c.String(200, map[bool]string{true: "deadline", false: "none"}[ok])
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, "deadline", w.Body.String())
}

func TestRateLimiter(t *testing.T) {
	current := time.Now()
	old := now
	now = func() time.Time { return current }
	defer func() { now = old }()

	r := newEngine(RateLimiter(2, time.Minute))
	assert.Equal(t, 200, call(r, "GET", nil).Code)
	assert.Equal(t, 200, call(r, "GET", nil).Code)
	assert.Equal(t, 429, call(r, "GET", nil).Code)
	current = current.Add(2 * time.Minute)
	assert.Equal(t, 200, call(r, "GET", nil).Code, "window baru harus mereset hitungan")
}
