// Package admin berisi layanan monitoring pengguna dan artikel untuk role admin.
package admin

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"app/internal/models"
)

// ErrNotFound dikembalikan ketika pengguna tidak ada.
var ErrNotFound = errors.New("pengguna tidak ditemukan")

// UserStat adalah identitas pengguna beserta statistik artikel.
type UserStat struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalPosts     int64     `json:"total_posts"`
	TotalPublished int64     `json:"total_published"`
}

// PostRow adalah ringkasan artikel (tanpa isi) untuk monitoring.
type PostRow struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Category    string     `json:"category"`
	IsPublished bool       `json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	AuthorID    uint       `json:"author_id"`
	Author      string     `json:"author"`
}

// UserPosts berisi identitas pengguna dan seluruh artikelnya.
type UserPosts struct {
	User  models.User `json:"user"`
	Posts []PostRow   `json:"posts"`
}

// Repository mengabstraksi query monitoring.
type Repository interface {
	ListUserStats() ([]UserStat, error)
	FindUser(id uint) (*models.User, error)
	ListPosts(userID uint) ([]PostRow, error)
}

// GormRepository adalah implementasi berbasis GORM.
type GormRepository struct{ db *gorm.DB }

// NewRepository membuat GormRepository.
func NewRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) ListUserStats() ([]UserStat, error) {
	out := []UserStat{}
	err := r.db.Table("users").
		Select("users.id, users.username, users.email, users.role, users.created_at, COUNT(posts.id) AS total_posts, COALESCE(SUM(CASE WHEN posts.is_published THEN 1 ELSE 0 END), 0) AS total_published").
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Group("users.id").Order("users.id").Scan(&out).Error
	return out, err
}

func (r *GormRepository) FindUser(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// ListPosts mengembalikan seluruh artikel; userID 0 berarti semua penulis.
func (r *GormRepository) ListPosts(userID uint) ([]PostRow, error) {
	out := []PostRow{}
	q := r.db.Table("posts").
		Select("posts.id, posts.title, posts.slug, posts.category, posts.is_published, posts.published_at, posts.created_at, posts.user_id AS author_id, users.username AS author").
		Joins("JOIN users ON users.id = posts.user_id")
	if userID != 0 {
		q = q.Where("posts.user_id = ?", userID)
	}
	err := q.Order("posts.id DESC").Scan(&out).Error
	return out, err
}

// Service berisi logika monitoring admin.
type Service struct{ repo Repository }

// NewService membuat Service admin.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListUsers mengembalikan seluruh pengguna beserta statistik artikel.
func (s *Service) ListUsers() ([]UserStat, error) { return s.repo.ListUserStats() }

// ListPosts mengembalikan arsip seluruh artikel dari semua penulis.
func (s *Service) ListPosts() ([]PostRow, error) { return s.repo.ListPosts(0) }

// ListUserPosts mengembalikan seluruh artikel milik pengguna tertentu.
func (s *Service) ListUserPosts(id uint) ([]PostRow, error) {
	posts, err := s.repo.ListPosts(id)
	if err != nil {
		return nil, err
	}
	if posts == nil {
		posts = []PostRow{}
	}
	return posts, nil
}
