package admin

import (
	"time"

	"gorm.io/gorm"
)

type UserStat struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalPosts     int64     `json:"total_posts"`
	TotalArticles  int64     `json:"total_articles"`
	TotalPublished int64     `json:"total_published"`
	TotalSelesai   int64     `json:"total_selesai"`
	BelumSelesai   int64     `json:"belum_selesai"`
}

type PostRow struct {
	ID          uint       `json:"id"`
	UserID      uint       `json:"user_id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Category    string     `json:"category"`
	IsPublished bool       `json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	AuthorID    uint       `json:"author_id"`
	Author      string     `json:"author"`
	AuthorName  string     `json:"author_name"`
}

type Repository interface {
	ListUserStats() ([]UserStat, error)
	ListPosts(userID uint) ([]PostRow, error)
}

type GormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) ListUserStats() ([]UserStat, error) {
	out := []UserStat{}
	err := r.db.Table("users").
		Select("users.id, users.username, users.email, users.role, users.created_at, COUNT(posts.id) AS total_articles, COALESCE(SUM(CASE WHEN posts.is_published THEN 1 ELSE 0 END), 0) AS total_published").
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Group("users.id, users.username, users.email, users.role, users.created_at").Order("users.id").Scan(&out).Error
	for i := range out {
		out[i].TotalPosts = out[i].TotalArticles
		out[i].TotalSelesai = out[i].TotalPublished
		out[i].BelumSelesai = out[i].TotalArticles - out[i].TotalPublished
	}
	return out, err
}

func (r *GormRepository) ListPosts(userID uint) ([]PostRow, error) {
	out := []PostRow{}
	q := r.db.Table("posts").
		Select("posts.id, posts.user_id, posts.title, posts.slug, posts.category, posts.is_published, posts.published_at, posts.created_at, posts.user_id AS author_id, users.username AS author, users.username AS author_name").
		Joins("JOIN users ON users.id = posts.user_id")
	if userID != 0 {
		q = q.Where("posts.user_id = ?", userID)
	}
	err := q.Order("posts.id DESC").Scan(&out).Error
	return out, err
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListUsers() ([]UserStat, error) { return s.repo.ListUserStats() }

func (s *Service) ListPosts() ([]PostRow, error) { return s.repo.ListPosts(0) }

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