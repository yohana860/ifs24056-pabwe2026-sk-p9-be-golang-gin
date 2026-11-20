// Package web menyajikan hasil build frontend React (folder dist) yang di-embed ke binary Go.
package web

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

//go:embed all:dist
var dist embed.FS

// FS mengembalikan filesystem frontend (isi folder dist).
func FS() fs.FS {
	sub, _ := fs.Sub(dist, "dist")
	return sub
}

// Handler menyajikan berkas statis; rute tanpa ekstensi di-fallback ke index.html (SPA).
func Handler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(path.Clean("/"+c.Request.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if st, err := fs.Stat(fsys, name); err != nil || st.IsDir() {
			if path.Ext(name) != "" {
				response.Fail(c, http.StatusNotFound, "Berkas tidak ditemukan")
				return
			}
			name = "index.html"
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			response.Fail(c, http.StatusNotFound, "Frontend belum di-build")
			return
		}
		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		if strings.HasPrefix(name, "assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		c.Data(http.StatusOK, contentType, data)
	}
}
