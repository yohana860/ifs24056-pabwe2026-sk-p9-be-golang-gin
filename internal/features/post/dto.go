package post

import "time"

// PostRequest adalah payload membuat/memperbarui artikel.
type PostRequest struct {
	Title          string `json:"title" binding:"required,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt" binding:"max=500"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path" binding:"max=2048"`
	IsPublished    bool   `json:"is_published"` // <-- WAJIB ADA UNTUK TEST SUITE 3
}

// Author adalah informasi publik penulis (tanpa email).
type Author struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// PostResponse adalah representasi artikel untuk klien.
type PostResponse struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content,omitempty"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	Author         *Author    `json:"author,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Filter adalah kriteria pencarian artikel publik.
type Filter struct {
	Q        string
	Category string
}

// Stats adalah ringkasan statistik penulis yang dihitung dari database.
type Stats struct {
	Total         int64 `json:"total"`
	TotalArticles int64 `json:"total_articles"`
	TotalTerbit   int64 `json:"total_terbit"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}