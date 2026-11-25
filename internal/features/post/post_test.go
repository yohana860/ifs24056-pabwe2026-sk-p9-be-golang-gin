package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/database"
	"app/internal/middleware"
	"app/internal/models"
)

const secret = "unit-test-secret-unit-test-secret-1234"

var seq atomic.Int64

func memDB(t *testing.T) *gorm.DB {
	db, err := database.OpenMemory(fmt.Sprintf("posttest%d", seq.Add(1)))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, db.Create(&[]models.User{
		{Username: "penulis1", Email: "p1@x.co", Password: "h", Role: "author"},
		{Username: "penulis2", Email: "p2@x.co", Password: "h", Role: "author"},
	}).Error)
	return db
}

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func engine(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(db)))
	r := gin.New()
	r.GET("/public/categories", h.Categories)
	r.GET("/public/posts", h.ListPublic)
	r.GET("/public/posts/:slug", h.GetPublic)
	g := r.Group("/posts", middleware.Auth(secret))
	g.GET("", h.ListMine)
	g.POST("", h.Create)
	g.GET("/stats", h.Stats)
	g.GET("/:id", h.GetMine)
	g.PUT("/:id", h.Update)
	g.PATCH("/:id/toggle-publish", h.TogglePublish)
	g.DELETE("/:id", h.Delete)
	return r
}

func call(r *gin.Engine, method, path string, uid uint, body any) (int, map[string]any) {
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if uid != 0 {
		tok, _ := middleware.GenerateToken(secret, uid, "u", "author", time.Hour)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func data(out map[string]any) map[string]any { return out["data"].(map[string]any) }

func newReq(title string) PostRequest {
	return PostRequest{Title: title, Category: "Teknologi", Excerpt: "ringkas", Content: "isi artikel"}
}

// TAMBAHAN: Helper untuk Update agar tidak error kompilasi
func newUpdateReq(title string) UpdatePostRequest {
	return UpdatePostRequest{Title: title, Category: "Teknologi", Excerpt: "ringkas", Content: "isi artikel"}
}

func TestSlugifyAndReadingTime(t *testing.T) {
	assert.Equal(t, "belajar-go-itu-asyik", slugify("  Belajar Go itu ASYIK!!  "))
	assert.Equal(t, "artikel", slugify("日本語 !!!"))
	assert.Len(t, slugify(strings.Repeat("a", 200)), 80)
	assert.Equal(t, strings.Repeat("a", 79), slugify(strings.Repeat("a", 79)+"-bbbb"), "trailing dash dipotong")
	assert.Equal(t, 1, readingTime(""))
	assert.Equal(t, 1, readingTime("satu dua tiga"))
	assert.Equal(t, 2, readingTime(strings.Repeat("kata ", 201)))
	assert.True(t, validCover(""))
	assert.True(t, validCover("https://a.test/x.png"))
	assert.True(t, validCover("http://a.test/x.png"))
	assert.True(t, validCover("/uploads/x.png"))
	assert.False(t, validCover("//evil.test/x.png"))
	assert.False(t, validCover("javascript:alert(1)"))
}

func TestAuthorCRUDAndOwnership(t *testing.T) {
	db := memDB(t)
	r := engine(db)

	code, out := call(r, "POST", "/posts", 1, newReq("Judul Sama"))
	require.Equal(t, 201, code)
	first := data(out)
	assert.Equal(t, "judul-sama", first["slug"])
	assert.Equal(t, false, first["is_published"])
	id := fmt.Sprint(int(first["id"].(float64)))

	code, out = call(r, "POST", "/posts", 1, newReq("Judul Sama"))
	require.Equal(t, 201, code)
	assert.Equal(t, "judul-sama-2", data(out)["slug"], "slug harus unik")
	code, out = call(r, "POST", "/posts", 2, newReq("Judul Sama"))
	assert.Equal(t, "judul-sama-3", data(out)["slug"])

	for _, bad := range []any{
		`{"title":""}`,
		PostRequest{Title: "x", Category: "Ngawur", Content: "c"},
		PostRequest{Title: "x", Category: "Umum"},
		PostRequest{Title: "x", Category: "Umum", Content: "c", CoverImagePath: "javascript:alert(1)"},
	} {
		code, _ = call(r, "POST", "/posts", 1, bad)
		assert.Equal(t, 400, code)
	}

	code, out = call(r, "GET", "/posts", 1, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, out["data"], 2, "hanya artikel milik sendiri")
	code, out = call(r, "GET", "/posts/"+id, 1, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "isi artikel", data(out)["content"])

	// IDOR: penulis lain tidak boleh membaca / mengubah / menghapus.
	for _, m := range []string{"GET", "PUT", "PATCH", "DELETE"} {
		path := "/posts/" + id
		var body any
		if m == "PUT" {
			body = newReq("retas")
		}
		if m == "PATCH" {
			path += "/toggle-publish"
		}
		code, _ = call(r, m, path, 2, body)
		assert.Equal(t, 403, code, m)
	}

	upd := newReq("Judul Baru")
	upd.CoverImagePath = "https://img.test/c.png"
	code, out = call(r, "PUT", "/posts/"+id, 1, upd)
	assert.Equal(t, 200, code)
	assert.Equal(t, "Judul Baru", data(out)["title"])
	assert.Equal(t, "judul-sama", data(out)["slug"], "slug stabil")
	code, _ = call(r, "PUT", "/posts/"+id, 1, PostRequest{Title: "x", Category: "Umum", Content: "c", CoverImagePath: "data:x"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/posts/"+id, 1, `{"title":""}`)
	assert.Equal(t, 400, code)

	code, _ = call(r, "DELETE", "/posts/"+id, 1, nil)
	assert.Equal(t, 200, code)
	code, _ = call(r, "DELETE", "/posts/"+id, 1, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "GET", "/posts/9999", 1, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "PUT", "/posts/9999", 1, newReq("x"))
	assert.Equal(t, 404, code)
	code, _ = call(r, "PATCH", "/posts/9999/toggle-publish", 1, nil)
	assert.Equal(t, 404, code)
	for _, m := range []string{"GET", "PUT", "DELETE"} {
		code, _ = call(r, m, "/posts/abc", 1, newReq("x"))
		assert.Equal(t, 400, code)
	}
	code, _ = call(r, "PATCH", "/posts/abc/toggle-publish", 1, nil)
	assert.Equal(t, 400, code)
	code, _ = call(r, "GET", "/posts", 0, nil)
	assert.Equal(t, 401, code)
}

func TestPublishFlowAndPublicVisibility(t *testing.T) {
	db := memDB(t)
	r := engine(db)
	_, out := call(r, "POST", "/posts", 1, PostRequest{Title: "Rahasia Draf", Category: "Opini", Excerpt: "e", Content: strings.Repeat("kata ", 450)})
	id := fmt.Sprint(int(data(out)["id"].(float64)))
	_, out = call(r, "POST", "/posts", 1, PostRequest{Title: "Tulisan Teknologi", Category: "Teknologi", Excerpt: "tentang golang", Content: "isi"})
	id2 := fmt.Sprint(int(data(out)["id"].(float64)))

	// draf tidak terlihat oleh guest.
	code, out := call(r, "GET", "/public/posts", 0, nil)
	assert.Equal(t, 200, code)
	assert.Empty(t, out["data"])
	code, _ = call(r, "GET", "/public/posts/rahasia-draf", 0, nil)
	assert.Equal(t, 404, code)

	code, out = call(r, "PATCH", "/posts/"+id+"/toggle-publish", 1, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, true, data(out)["is_published"])
	assert.NotNil(t, data(out)["published_at"])
	call(r, "PATCH", "/posts/"+id2+"/toggle-publish", 1, nil)

	code, out = call(r, "GET", "/public/posts/rahasia-draf", 0, nil)
	assert.Equal(t, 200, code)
	d := data(out)
	assert.EqualValues(t, 3, d["reading_time"])
	assert.NotEmpty(t, d["content"])
	assert.Equal(t, "penulis1", d["author"].(map[string]any)["username"])
	assert.NotContains(t, fmt.Sprint(out), "p1@x.co", "email penulis tidak bocor")

	_, out = call(r, "GET", "/public/posts", 0, nil)
	list := out["data"].([]any)
	assert.Len(t, list, 2)
	assert.Nil(t, list[0].(map[string]any)["content"], "daftar tidak membawa isi penuh")
	_, out = call(r, "GET", "/public/posts?category=Opini", 0, nil)
	assert.Len(t, out["data"], 1)
	_, out = call(r, "GET", "/public/posts?category=Semua", 0, nil)
	assert.Len(t, out["data"], 2)
	_, out = call(r, "GET", "/public/posts?q=GOLANG", 0, nil)
	assert.Len(t, out["data"], 1)
	_, out = call(r, "GET", "/public/posts?q=%25", 0, nil)
	assert.Empty(t, out["data"], "wildcard di-escape")

	// kembali ke draf: published_at dikosongkan & hilang dari publik.
	code, out = call(r, "PATCH", "/posts/"+id+"/toggle-publish", 1, nil)
	assert.Equal(t, false, data(out)["is_published"])
	assert.Nil(t, data(out)["published_at"])
	code, _ = call(r, "GET", "/public/posts/rahasia-draf", 0, nil)
	assert.Equal(t, 404, code)

	_, out = call(r, "GET", "/posts/stats", 1, nil)
	st := data(out)
	assert.EqualValues(t, 2, st["total"])
	assert.EqualValues(t, 1, st["total_terbit"])
	assert.EqualValues(t, 1, st["belum_selesai"])

	_, out = call(r, "GET", "/public/categories", 0, nil)
	assert.Equal(t, []any{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}, out["data"])
}

func TestRepositoryErrorsOnClosedDB(t *testing.T) {
	db := memDB(t)
	repo := NewRepository(db)
	closeDB(db)
	_, err := repo.ListPublished(Filter{Q: "a", Category: "Umum"})
	assert.Error(t, err)
	_, err = repo.FindPublishedBySlug("x")
	assert.Error(t, err)
	_, err = repo.ListByUser(1)
	assert.Error(t, err)
	_, err = repo.FindOwned(1, 1)
	assert.Error(t, err)
	assert.Error(t, repo.Create(&models.Post{}))
	assert.Error(t, repo.Save(&models.Post{ID: 1}))
	assert.Error(t, repo.Delete(1, 1))
	_, err = repo.SlugExists("x")
	assert.Error(t, err)
	_, _, err = repo.Count(1)
	assert.Error(t, err)
}

func TestCountSecondQueryFails(t *testing.T) {
	db := memDB(t)
	calls := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("fail_second", func(tx *gorm.DB) {
		calls++
		if calls == 2 {
			tx.AddError(errors.New("query kedua gagal"))
		}
	}))
	_, _, err := NewRepository(db).Count(1)
	assert.Error(t, err)
}

type wrapRepo struct {
	Repository
	slugErr, saveErr, createErr, countErr error
}

func (w wrapRepo) SlugExists(s string) (bool, error) {
	if w.slugErr != nil {
		return false, w.slugErr
	}
	return w.Repository.SlugExists(s)
}
func (w wrapRepo) Save(p *models.Post) error {
	if w.saveErr != nil {
		return w.saveErr
	}
	return w.Repository.Save(p)
}
func (w wrapRepo) Create(p *models.Post) error {
	if w.createErr != nil {
		return w.createErr
	}
	return w.Repository.Create(p)
}
func (w wrapRepo) Count(u uint) (int64, int64, error) {
	if w.countErr != nil {
		return 0, 0, w.countErr
	}
	return w.Repository.Count(u)
}

func TestServiceErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	db := memDB(t)
	base := NewRepository(db)
	_, err := NewService(base).Create(1, newReq("Ada"))
	require.NoError(t, err)
	svc := func(w wrapRepo) *Service { w.Repository = base; return NewService(w) }

	_, err = svc(wrapRepo{slugErr: boom}).Create(1, newReq("x"))
	assert.ErrorIs(t, err, boom)
	_, err = svc(wrapRepo{createErr: boom}).Create(1, newReq("x"))
	assert.ErrorIs(t, err, boom)
	
	// PERBAIKAN: Menggunakan newUpdateReq
	_, err = svc(wrapRepo{saveErr: boom}).Update(1, 1, newUpdateReq("x"))
	assert.ErrorIs(t, err, boom)
	_, err = svc(wrapRepo{}).Update(1, 99, newUpdateReq("x"))
	assert.ErrorIs(t, err, ErrNotFound)
	
	_, err = svc(wrapRepo{saveErr: boom}).TogglePublish(1, 1)
	assert.ErrorIs(t, err, boom)
	_, err = svc(wrapRepo{countErr: boom}).Stats(1)
	assert.ErrorIs(t, err, boom)
}

func TestHandlerServerErrors(t *testing.T) {
	db := memDB(t)
	r := engine(db)
	closeDB(db)
	for _, tc := range []struct {
		m, p string
		b    any
	}{
		{"GET", "/public/posts", nil}, {"GET", "/public/posts/x", nil}, {"GET", "/posts", nil}, {"GET", "/posts/stats", nil},
		{"GET", "/posts/1", nil}, {"POST", "/posts", newReq("x")}, {"PUT", "/posts/1", newReq("x")},
		{"PATCH", "/posts/1/toggle-publish", nil}, {"DELETE", "/posts/1", nil},
	} {
		code, out := call(r, tc.m, tc.p, 1, tc.b)
		assert.Equal(t, 500, code, tc.m+" "+tc.p)
		assert.Equal(t, "error", out["status"])
	}
}

func TestDefaultCategory(t *testing.T) {
	r := engine(memDB(t))
	code, out := call(r, "POST", "/posts", 1, PostRequest{Title: "Tanpa Kategori", Content: "konten"})
	assert.Equal(t, 201, code)
	assert.Equal(t, "Umum", data(out)["category"])
	assert.Nil(t, data(out)["published_at"])

	id := fmt.Sprint(int(data(out)["id"].(float64)))
	code, out = call(r, "PUT", "/posts/"+id, 1, PostRequest{Title: "Judul Update", Content: "konten baru", Category: ""})
	assert.Equal(t, 200, code)
	assert.Equal(t, "Umum", data(out)["category"])

	code, _ = call(r, "PUT", "/posts/"+id, 1, PostRequest{Title: "Judul Update", Content: "konten baru", Category: "Ngawur"})
	assert.Equal(t, 400, code)
}