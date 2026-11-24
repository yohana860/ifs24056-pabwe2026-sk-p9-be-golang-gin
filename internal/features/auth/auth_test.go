package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	db, err := database.OpenMemory(fmt.Sprintf("authtest%d", seq.Add(1)))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func newEngine(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(db), secret, time.Hour))
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	g := r.Group("/", middleware.Auth(secret))
	g.POST("/logout", h.Logout)
	g.GET("/profile", h.Profile)
	g.PUT("/profile", h.UpdateProfile)
	return r
}

func send(r *gin.Engine, method, path, token string, body any) (*httptest.ResponseRecorder, map[string]any) {
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func register(t *testing.T, r *gin.Engine, name string) {
	w, _ := send(r, "POST", "/register", "", RegisterRequest{Username: name, Email: name + "@mail.test", Password: "secret123"})
	require.Equal(t, 201, w.Code)
}

func login(t *testing.T, r *gin.Engine, name string) string {
	w, out := send(r, "POST", "/login", "", LoginRequest{Username: name, Password: "secret123"})
	require.Equal(t, 200, w.Code)
	return out["data"].(map[string]any)["token"].(string)
}

func TestRegisterLoginFlow(t *testing.T) {
	db := memDB(t)
	r := newEngine(db)

	w, out := send(r, "POST", "/register", "", map[string]any{"username": "yobez", "email": "Yobez@Mail.Test", "password": "secret123", "role": "admin"})
	assert.Equal(t, 201, w.Code)
	user := out["data"].(map[string]any)
	assert.Equal(t, "author", user["role"], "role tidak boleh dipercaya dari klien")
	assert.Equal(t, "yobez@mail.test", user["email"])
	assert.NotContains(t, w.Body.String(), "password")

	var stored models.User
	require.NoError(t, db.First(&stored).Error)
	assert.NotEqual(t, "secret123", stored.Password)

	w, _ = send(r, "POST", "/register", "", RegisterRequest{Username: "yobez", Email: "other@mail.test", Password: "secret123"})
	assert.Equal(t, 400, w.Code)
	w, _ = send(r, "POST", "/register", "", RegisterRequest{Username: "short", Password: "123"})
	assert.Equal(t, 400, w.Code)

	token := login(t, r, "yobez")
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, login(t, r, "yobez@mail.test"), "login dengan email")

	w, _ = send(r, "POST", "/login", "", LoginRequest{Username: "yobez", Password: "salah"})
	assert.Equal(t, 401, w.Code)
	w, _ = send(r, "POST", "/login", "", LoginRequest{Username: "tidakada", Password: "secret123"})
	assert.Equal(t, 401, w.Code)
	w, _ = send(r, "POST", "/login", "", `{"username":""}`)
	assert.Equal(t, 400, w.Code)

	w, _ = send(r, "POST", "/logout", token, nil)
	assert.Equal(t, 200, w.Code)
	w, _ = send(r, "POST", "/logout", "", nil)
	assert.Equal(t, 401, w.Code)
}

func TestProfile(t *testing.T) {
	db := memDB(t)
	r := newEngine(db)
	register(t, r, "alice")
	register(t, r, "bob")
	token := login(t, r, "alice")

	w, out := send(r, "GET", "/profile", token, nil)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "alice", out["data"].(map[string]any)["username"])

	w, out = send(r, "PUT", "/profile", token, UpdateProfileRequest{Username: "alice2", Email: "A2@mail.test", Password: "newpass123"})
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "alice2", out["data"].(map[string]any)["username"])
	w, _ = send(r, "POST", "/login", "", LoginRequest{Username: "alice2", Password: "newpass123"})
	assert.Equal(t, 200, w.Code)

	w, _ = send(r, "PUT", "/profile", token, UpdateProfileRequest{Username: "bob"})
	assert.Equal(t, 409, w.Code)
	w, _ = send(r, "PUT", "/profile", token, `{"email":"bukan-email"}`)
	assert.Equal(t, 400, w.Code)
	w, _ = send(r, "PUT", "/profile", token, UpdateProfileRequest{})
	assert.Equal(t, 200, w.Code, "payload kosong tidak mengubah apa pun")

	require.NoError(t, db.Where("username = ?", "alice2").Delete(&models.User{}).Error)
	w, _ = send(r, "GET", "/profile", token, nil)
	assert.Equal(t, 404, w.Code)

	closeDB(db)
	w, _ = send(r, "GET", "/profile", token, nil)
	assert.Equal(t, 500, w.Code)
	w, _ = send(r, "POST", "/register", "", RegisterRequest{Username: "zzz", Email: "z@mail.test", Password: "secret123"})
	assert.Equal(t, 500, w.Code)
}

func TestRepositoryErrorsOnClosedDB(t *testing.T) {
	db := memDB(t)
	repo := NewRepository(db)
	closeDB(db)
	assert.Error(t, repo.Create(&models.User{Username: "a", Email: "a", Password: "x"}))
	_, err := repo.FindByID(1)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	_, err = repo.FindByLogin("a")
	assert.Error(t, err)
	_, err = repo.ExistsOther("a", "a", 0)
	assert.Error(t, err)
	assert.Error(t, repo.Save(&models.User{ID: 1}))
}

func TestRepositoryNotFound(t *testing.T) {
	repo := NewRepository(memDB(t))
	_, err := repo.FindByID(99)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = repo.FindByLogin("nobody")
	assert.ErrorIs(t, err, ErrNotFound)
	exists, err := repo.ExistsOther("a", "a", 0)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// fakeRepo memungkinkan injeksi kegagalan pada langkah tertentu di service.
type fakeRepo struct {
	user                          *models.User
	existsErr, createErr, saveErr error
	findErr, loginErr             error
	taken                         bool
}

func (f *fakeRepo) Create(*models.User) error                      { return f.createErr }
func (f *fakeRepo) FindByID(uint) (*models.User, error)            { return f.user, f.findErr }
func (f *fakeRepo) FindByLogin(string) (*models.User, error)       { return f.user, f.loginErr }
func (f *fakeRepo) ExistsOther(string, string, uint) (bool, error) { return f.taken, f.existsErr }
func (f *fakeRepo) Save(*models.User) error                        { return f.saveErr }

func TestServiceErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	long := strings.Repeat("a", 80)
	reg := RegisterRequest{Username: "abc", Email: "a@b.co", Password: "secret123"}

	_, err := NewService(&fakeRepo{existsErr: boom}, secret, time.Hour).Register(reg)
	assert.ErrorIs(t, err, boom)
	_, err = NewService(&fakeRepo{}, secret, time.Hour).Register(RegisterRequest{Username: "abc", Email: "a@b.co", Password: long})
	assert.Error(t, err, "bcrypt menolak password > 72 byte")
	_, err = NewService(&fakeRepo{createErr: boom}, secret, time.Hour).Register(reg)
	assert.ErrorIs(t, err, boom)

	_, err = NewService(&fakeRepo{loginErr: boom}, secret, time.Hour).Login(LoginRequest{Username: "a", Password: "b"})
	assert.ErrorIs(t, err, boom)

	u := func() *models.User { return &models.User{ID: 1, Username: "u", Email: "u@u.co", Password: "x"} }
	_, err = NewService(&fakeRepo{findErr: boom}, secret, time.Hour).UpdateProfile(1, UpdateProfileRequest{})
	assert.ErrorIs(t, err, boom)
	_, err = NewService(&fakeRepo{user: u(), existsErr: boom}, secret, time.Hour).UpdateProfile(1, UpdateProfileRequest{})
	assert.ErrorIs(t, err, boom)
	_, err = NewService(&fakeRepo{user: u()}, secret, time.Hour).UpdateProfile(1, UpdateProfileRequest{Password: long})
	assert.Error(t, err)
	_, err = NewService(&fakeRepo{user: u(), saveErr: boom}, secret, time.Hour).UpdateProfile(1, UpdateProfileRequest{})
	assert.ErrorIs(t, err, boom)
	_, err = NewService(&fakeRepo{findErr: boom}, secret, time.Hour).Profile(1)
	assert.ErrorIs(t, err, boom)
}

func TestHandlerFailMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := map[error]int{ErrConflict: 409, ErrInvalidCredentials: 401, ErrNotFound: 404, errors.New("x"): 500}
	for err, code := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		fail(c, err)
		assert.Equal(t, code, w.Code)
	}
	_ = http.StatusOK
}
