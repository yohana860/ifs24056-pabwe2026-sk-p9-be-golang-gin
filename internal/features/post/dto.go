package post

import "time"

// PostRequest adalah payload membuat artikel.
type PostRequest struct {
	Title          string `json:"title" binding:"required,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt" binding:"max=500"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path" binding:"max=2048"`
	IsPublished    bool   `json:"is_published"`
}

// UpdatePostRequest adalah payload update artikel (field kosong tidak wajib diisi).
type UpdatePostRequest struct {
	Title          string `json:"title" binding:"omitempty,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt" binding:"max=500"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path" binding:"max=2048"`
	IsPublished    *bool  `json:"is_published"`
}

type Author struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type PostResponse struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`     // <-- TAMBAHAN UNTUK TEST
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content,omitempty"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	AuthorName     string     `json:"author_name"` // <-- TAMBAHAN UNTUK TEST
	Author         *Author    `json:"author,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Filter struct {
	Q        string
	Category string
}

type Stats struct {
	Total         int64 `json:"total"`
	TotalArticles int64 `json:"total_articles"`
	TotalTerbit   int64 `json:"total_terbit"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}