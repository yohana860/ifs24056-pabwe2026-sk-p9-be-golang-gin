package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/database"
	"app/internal/features/ai"
	"app/internal/web"
)

var seq atomic.Int64

type chatFn func() (string, error)

func (f chatFn) Chat(context.Context, []ai.Message) (string, error) { return f() }

type searchFn func() ([]ai.SearchResult, error)

func (f searchFn) Search(context.Context, string) ([]ai.SearchResult, error) { return f() }

func testCfg() *config.Config {
	return &config.Config{AppEnv: "development", AppPort: "0", AppKey: "unit-test-secret-unit-test-secret-1234",
		CORSOrigins: []string{"*"}, AdminUsername: "admin", AdminPassword: "admin"}
}

func newDB(t *testing.T) *gorm.DB {
	db, err := database.OpenMemory(fmt.Sprintf("routertest%d", seq.Add(1)))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, database.Seed(db, testCfg()))
	return db
}

func newApp(t *testing.T, chat ai.ChatClient, search ai.Searcher, spa gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(testCfg(), newDB(t), chat, search, spa)
}

func req(r *gin.Engine, method, path, token string, body any) (*httptest.ResponseRecorder, map[string]any) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	rq := httptest.NewRequest(method, path, &buf)
	rq.Header.Set("Content-Type", "application/json")
	if token != "" {
		rq.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, rq)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func login(t *testing.T, r *gin.Engine, user, pass string) string {
	w, out := req(r, "POST", "/api/auth/login", "", map[string]string{"username": user, "password": pass})
	require.Equal(t, 200, w.Code)
	return out["data"].(map[string]any)["token"].(string)
}

func TestEndToEndBlogWorkflow(t *testing.T) {
	r := newApp(t, nil, nil, nil)
	author := login(t, r, "author", "author")
	admin := login(t, r, "admin", "admin")

	w, out := req(r, "POST", "/api/auth/register", "", map[string]string{"username": "penulis", "email": "p@x.co", "password": "rahasia1"})
	require.Equal(t, 201, w.Code)
	assert.Equal(t, "author", out["data"].(map[string]any)["role"])
	baru := login(t, r, "penulis", "rahasia1")

	w, out = req(r, "POST", "/api/posts", author, map[string]string{"title": "Halo Dunia", "category": "Umum", "content": "# Isi"})
	require.Equal(t, 201, w.Code)
	id := fmt.Sprint(int(out["data"].(map[string]any)["id"].(float64)))

	w, _ = req(r, "GET", "/api/public/posts/halo-dunia", "", nil)
	assert.Equal(t, 404, w.Code, "draf tidak terlihat guest")
	req(r, "PATCH", "/api/posts/"+id+"/toggle-publish", author, nil)
	w, _ = req(r, "GET", "/api/public/posts/halo-dunia", "", nil)
	assert.Equal(t, 200, w.Code)
	w, _ = req(r, "GET", "/api/public/categories", "", nil)
	assert.Equal(t, 200, w.Code)
	w, _ = req(r, "GET", "/api/posts/"+id, baru, nil)
	assert.Equal(t, 403, w.Code, "penulis lain tidak boleh mengelola artikel ini")

	// role guard: admin hanya monitoring, author tidak boleh ke /admin.
	w, _ = req(r, "POST", "/api/posts", admin, map[string]string{"title": "x", "category": "Umum", "content": "c"})
	assert.Equal(t, 403, w.Code)
	w, _ = req(r, "GET", "/api/admin/users", author, nil)
	assert.Equal(t, 403, w.Code)
	w, _ = req(r, "GET", "/api/admin/users", "", nil)
	assert.Equal(t, 401, w.Code)
	w, out = req(r, "GET", "/api/admin/users", admin, nil)
	assert.Equal(t, 200, w.Code)
	assert.Len(t, out["data"], 4)
	w, out = req(r, "GET", "/api/admin/posts", admin, nil)
	assert.Equal(t, 200, w.Code)
	assert.Len(t, out["data"], 1)
	w, _ = req(r, "GET", "/api/admin/users/2/posts", admin, nil)
	assert.Equal(t, 200, w.Code)
}

func TestAIGenerateEndToEnd(t *testing.T) {
	draft := `{"title":"Berita Go","category":"Teknologi","excerpt":"e","content":"isi [1]"}`
	src := []ai.SearchResult{{Title: "Rilis Go", URL: "https://go.dev/blog", Snippet: "s"}}
	r := newApp(t, chatFn(func() (string, error) { return draft, nil }), searchFn(func() ([]ai.SearchResult, error) { return src, nil }), nil)
	tok := login(t, r, "author", "author")
	w, out := req(r, "POST", "/api/posts/ai-generate", tok, map[string]string{"topic": "Rilis Go terbaru"})
	assert.Equal(t, 200, w.Code)
	d := out["data"].(map[string]any)
	assert.Equal(t, true, d["searched"])
	assert.Equal(t, "https://go.dev/blog", d["sources"].([]any)[0].(map[string]any)["url"])

	r2 := newApp(t, chatFn(func() (string, error) { return draft, nil }), searchFn(func() ([]ai.SearchResult, error) { return nil, ai.ErrSearchFailed }), nil)
	w, out = req(r2, "POST", "/api/posts/ai-generate", login(t, r2, "author", "author"), map[string]string{"topic": "Rilis Go terbaru"})
	assert.Equal(t, 502, w.Code)
	assert.Equal(t, "error", out["status"])
}

func TestAuthRateLimit(t *testing.T) {
	r := newApp(t, nil, nil, nil)
	var last int
	for i := 0; i < 21; i++ {
		w, _ := req(r, "POST", "/api/auth/login", "", map[string]string{"username": "x", "password": "y"})
		last = w.Code
	}
	assert.Equal(t, 429, last)
}

func TestNoRouteAndSPA(t *testing.T) {
	fsys := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}, "assets/a.js": {Data: []byte("1")}}
	r := newApp(t, nil, nil, web.Handler(fsys))

	w, out := req(r, "GET", "/api/tidak-ada", "", nil)
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "Endpoint not found", out["message"], "API tidak boleh tertangkap SPA fallback")
	w, _ = req(r, "GET", "/swagger-lain", "", nil)
	assert.Equal(t, 404, w.Code)
	w, _ = req(r, "POST", "/halaman", "", nil)
	assert.Equal(t, 404, w.Code)
	for _, p := range []string{"/", "/author", "/admin", "/posts/slug-artikel", "/login"} {
		w, _ = req(r, "GET", p, "", nil)
		assert.Equal(t, 200, w.Code, p)
		assert.Contains(t, w.Body.String(), "spa", p)
	}
	w, _ = req(r, "GET", "/assets/a.js", "", nil)
	assert.Equal(t, 200, w.Code)
	w, _ = req(r, "HEAD", "/author", "", nil)
	assert.Equal(t, 200, w.Code)

	api := newApp(t, nil, nil, nil)
	w, _ = req(api, "GET", "/author", "", nil)
	assert.Equal(t, 404, w.Code)
}

func TestSwaggerAndSpec(t *testing.T) {
	r := newApp(t, nil, nil, nil)
	w, _ := req(r, "GET", "/swagger/index.html", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "swagger")

	w, out := req(r, "GET", "/api.json", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "2.0", out["swagger"])
	assert.NotEmpty(t, out["paths"])
	assert.Equal(t, []any{"http"}, out["schemes"])

	rq := httptest.NewRequest("GET", "/api.json", nil)
	rq.Header.Set("X-Forwarded-Proto", "https")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, rq)
	assert.Contains(t, w.Body.String(), `"https"`)

	old := swaggerJSON
	swaggerJSON = func(string, string) ([]byte, error) { return nil, errors.New("rusak") }
	defer func() { swaggerJSON = old }()
	w, _ = req(r, "GET", "/api.json", "", nil)
	assert.Equal(t, 500, w.Code)
}

func TestPanicRecoveryAndReleaseMode(t *testing.T) {
	cfg := testCfg()
	cfg.AppEnv = "production"
	r := NewRouter(cfg, newDB(t), nil, nil, nil)
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
	r.GET("/panic", func(*gin.Context) { panic("sengaja") })
	w, out := req(r, "GET", "/panic", "", nil)
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "error", out["status"])
	assert.NotContains(t, w.Body.String(), "sengaja")
}

func TestRun(t *testing.T) {
	var gotAddr string
	var gotHandler http.Handler
	listen := func(addr string, h http.Handler) error { gotAddr, gotHandler = addr, h; return nil }
	good := func(*config.Config) (*gorm.DB, error) { return newDB(t), nil }

	cfg := testCfg()
	cfg.AppPort = "8081"
	require.NoError(t, Run(cfg, good, listen))
	assert.Equal(t, ":8081", gotAddr)
	assert.NotNil(t, gotHandler)

	prod := testCfg()
	prod.AppEnv, prod.AppKey = "production", "pendek"
	assert.Error(t, Run(prod, good, listen))
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })

	boom := errors.New("tidak bisa konek")
	assert.ErrorIs(t, Run(testCfg(), func(*config.Config) (*gorm.DB, error) { return nil, boom }, listen), boom)

	closedDB := func(*config.Config) (*gorm.DB, error) {
		db := newDB(t)
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		return db, nil
	}
	assert.Error(t, Run(testCfg(), closedDB, listen))

	seedFail := func(*config.Config) (*gorm.DB, error) {
		db, err := database.OpenMemory(fmt.Sprintf("routertest%d", seq.Add(1)))
		require.NoError(t, err)
		require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail", func(tx *gorm.DB) {
			tx.AddError(errors.New("insert gagal"))
		}))
		return db, nil
	}
	assert.Error(t, Run(testCfg(), seedFail, listen))

	assert.ErrorIs(t, Run(testCfg(), good, func(string, http.Handler) error { return boom }), boom)
}
