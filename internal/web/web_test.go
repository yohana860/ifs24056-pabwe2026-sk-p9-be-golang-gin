package web

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func serve(fsys fs.FS, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.NoRoute(Handler(fsys))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestHandlerServesAssetsAndFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":      {Data: []byte("<html>app</html>")},
		"assets/app.js":   {Data: []byte("console.log(1)")},
		"assets/app.css":  {Data: []byte("body{}")},
		"data.unknownext": {Data: []byte("plain text here")},
		"folder/x.txt":    {Data: []byte("x")},
	}
	w := serve(fsys, "/")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))

	for _, p := range []string{"/author", "/admin", "/posts/hello-world", "/todos/5", "/folder"} {
		w = serve(fsys, p)
		assert.Equal(t, 200, w.Code, p)
		assert.Equal(t, "<html>app</html>", w.Body.String(), p)
	}

	w = serve(fsys, "/assets/app.js")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "javascript")
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	w = serve(fsys, "/assets/app.css")
	assert.Contains(t, w.Header().Get("Content-Type"), "text/css")

	w = serve(fsys, "/data.unknownext")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")

	w = serve(fsys, "/assets/hilang.js")
	assert.Equal(t, 404, w.Code, "aset hilang tidak boleh jatuh ke index.html")
	w = serve(fsys, "/../../etc/passwd")
	assert.Equal(t, "<html>app</html>", w.Body.String(), "path traversal dinormalisasi, tidak membaca di luar dist")
}

func TestHandlerWithoutBuild(t *testing.T) {
	w := serve(fstest.MapFS{}, "/author")
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "belum di-build")
}

type brokenFS struct{}

func (brokenFS) Open(string) (fs.File, error) { return nil, errors.New("rusak") }
func (brokenFS) Stat(string) (fs.FileInfo, error) {
	return fstest.MapFS{"index.html": {Data: []byte("x")}}.Stat("index.html")
}

func TestHandlerReadFailure(t *testing.T) {
	w := serve(brokenFS{}, "/")
	assert.Equal(t, 404, w.Code)
}

func TestEmbeddedFS(t *testing.T) {
	// folder dist hanya berisi .gitkeep sebelum frontend di-build.
	_, err := fs.Stat(FS(), ".gitkeep")
	assert.NoError(t, err)
}
