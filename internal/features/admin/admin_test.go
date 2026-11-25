package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/database"
	"app/internal/models"
)

var seq atomic.Int64

func memDB(t *testing.T) *gorm.DB {
	db, err := database.OpenMemory(fmt.Sprintf("admintest%d", seq.Add(1)))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, db.Create(&[]models.User{
		{Username: "a", Email: "a@x.co", Password: "h", Role: "admin"},
		{Username: "b", Email: "b@x.co", Password: "h", Role: "author"},
	}).Error)
	require.NoError(t, db.Create(&[]models.Post{
		{UserID: 2, Title: "P1", Slug: "p1", Category: "Umum", Content: "c", IsPublished: true},
		{UserID: 2, Title: "P2", Slug: "p2", Category: "Umum", Content: "c"},
		{UserID: 2, Title: "P3", Slug: "p3", Category: "Umum", Content: "c", IsPublished: true},
	}).Error)
	return db
}

func engine(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(db)))
	r := gin.New()
	r.GET("/users", h.ListUsers)
	r.GET("/posts", h.ListPosts)
	r.GET("/users/:id/posts", h.ListUserPosts)
	return r
}

func get(r *gin.Engine, path string) (int, map[string]any) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestMonitoring(t *testing.T) {
	r := engine(memDB(t))
	code, out := get(r, "/users")
	assert.Equal(t, 200, code)
	users := out["data"].([]any)
	require.Len(t, users, 2)
	b := users[1].(map[string]any)
	
	assert.EqualValues(t, 3, b["total_articles"])
	assert.EqualValues(t, 3, b["total_posts"])
	assert.EqualValues(t, 2, b["total_published"])
	assert.EqualValues(t, 2, b["total_selesai"])
	assert.EqualValues(t, 1, b["belum_selesai"])
	assert.EqualValues(t, 0, users[0].(map[string]any)["total_articles"])
	assert.NotContains(t, fmt.Sprint(out), "password")

	code, out = get(r, "/posts")
	assert.Equal(t, 200, code)
	posts := out["data"].([]any)
	require.Len(t, posts, 3)
	assert.Equal(t, "b", posts[0].(map[string]any)["author"])
	assert.Equal(t, "b", posts[0].(map[string]any)["author_name"])
	assert.EqualValues(t, 2, posts[0].(map[string]any)["user_id"])
	assert.NotContains(t, fmt.Sprint(out), "content")

	code, out = get(r, "/users/2/posts")
	assert.Equal(t, 200, code)
	assert.Len(t, out["data"].([]any), 3)
	code, out = get(r, "/users/1/posts")
	assert.Equal(t, 200, code)
	assert.Empty(t, out["data"].([]any))
	code, out = get(r, "/users/99/posts")
	assert.Equal(t, 200, code)
	assert.Empty(t, out["data"].([]any))
	code, _ = get(r, "/users/abc/posts")
	assert.Equal(t, 400, code)
}

func TestErrorsOnClosedDB(t *testing.T) {
	db := memDB(t)
	r := engine(db)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	for _, p := range []string{"/users", "/posts", "/users/1/posts"} {
		code, _ := get(r, p)
		assert.Equal(t, 500, code, p)
	}
}

type stubRepo struct{ postsErr error }

func (s stubRepo) ListUserStats() ([]UserStat, error) { return nil, nil }
func (s stubRepo) ListPosts(uint) ([]PostRow, error)  { return nil, s.postsErr }

func TestServicePostsError(t *testing.T) {
	_, err := NewService(stubRepo{postsErr: errors.New("x")}).ListUserPosts(1)
	assert.Error(t, err)
	posts, err := NewService(stubRepo{}).ListUserPosts(1)
	require.NoError(t, err)
	assert.NotNil(t, posts)
	assert.Empty(t, posts)
}