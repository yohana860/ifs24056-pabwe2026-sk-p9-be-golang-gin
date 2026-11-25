package post

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"app/internal/models"
)

var ErrInvalidCover = errors.New("cover_image_path harus berupa URL http(s) atau path yang diawali /")
var ErrInvalidCategory = errors.New("kategori tidak valid")

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Categories() []string { return models.Categories }

func slugify(title string) string {
	var b strings.Builder
	dash := true
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	if slug == "" {
		return "artikel"
	}
	return slug
}

func readingTime(content string) int {
	minutes := (len(strings.Fields(content)) + 199) / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

func validCover(c string) bool {
	return c == "" || strings.HasPrefix(c, "https://") || strings.HasPrefix(c, "http://") ||
		(strings.HasPrefix(c, "/") && !strings.HasPrefix(c, "//"))
}

func toResponse(p *models.Post, withContent bool) PostResponse {
	res := PostResponse{
		ID: p.ID, UserID: p.UserID, Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		CoverImagePath: p.CoverImagePath, IsPublished: p.IsPublished, PublishedAt: p.PublishedAt,
		ReadingTime: readingTime(p.Content), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if withContent {
		res.Content = p.Content
	}
	if p.User != nil {
		res.AuthorName = p.User.Username
		res.Author = &Author{ID: p.User.ID, Username: p.User.Username}
	}
	return res
}

func toList(posts []models.Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, toResponse(&posts[i], false))
	}
	return out
}

func (s *Service) ListPublic(f Filter) ([]PostResponse, error) {
	f.Q = strings.TrimSpace(f.Q)
	if f.Category == "Semua" {
		f.Category = ""
	}
	posts, err := s.repo.ListPublished(f)
	if err != nil {
		return nil, err
	}
	out := toList(posts)
	for i := range out {
		out[i].UserID = 0 // data internal: jangan tampil di API publik
	}
	return out, nil
}

func (s *Service) GetPublic(slug string) (*PostResponse, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	if err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	res.UserID = 0 // data internal: jangan tampil di API publik
	return &res, nil
}

func (s *Service) ListMine(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	return toList(posts), nil
}

func (s *Service) GetMine(userID, id uint) (*PostResponse, error) {
	p, err := s.repo.FindOwned(userID, id)
	if err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

func (s *Service) uniqueSlug(title string) (string, error) {
	base := slugify(title)
	slug := base
	for i := 2; i < 1000; i++ {
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return "", errors.New("gagal membuat slug unik")
}

func (s *Service) Create(userID uint, req PostRequest) (*PostResponse, error) {
	if !validCover(req.CoverImagePath) {
		return nil, ErrInvalidCover
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}
	if !slices.Contains(models.Categories, category) {
		return nil, ErrInvalidCategory
	}
	slug, err := s.uniqueSlug(req.Title)
	if err != nil {
		return nil, err
	}

	p := &models.Post{
		UserID: userID, Title: strings.TrimSpace(req.Title), Slug: slug, Category: category,
		Excerpt: req.Excerpt, Content: req.Content, CoverImagePath: req.CoverImagePath,
		IsPublished: req.IsPublished,
	}

	if p.IsPublished {
		now := s.now()
		p.PublishedAt = &now
	}

	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

func (s *Service) Update(userID, id uint, req UpdatePostRequest) (*PostResponse, error) {
	if !validCover(req.CoverImagePath) {
		return nil, ErrInvalidCover
	}

	p, err := s.repo.FindOwned(userID, id)
	if err != nil {
		return nil, err
	}

	// Update parsial (hanya ubah jika field tidak kosong)
	if req.Title != "" {
		p.Title = strings.TrimSpace(req.Title)
	}
	if req.Category != "" {
		category := strings.TrimSpace(req.Category)
		if !slices.Contains(models.Categories, category) {
			return nil, ErrInvalidCategory
		}
		p.Category = category
	}
	if req.Excerpt != "" {
		p.Excerpt = req.Excerpt
	}
	if req.Content != "" {
		p.Content = req.Content
	}
	if req.CoverImagePath != "" {
		p.CoverImagePath = req.CoverImagePath
	}
	if req.IsPublished != nil {
		p.IsPublished = *req.IsPublished
		if p.IsPublished && p.PublishedAt == nil {
			now := s.now()
			p.PublishedAt = &now
		} else if !p.IsPublished {
			p.PublishedAt = nil
		}
	}

	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

func (s *Service) TogglePublish(userID, id uint) (*PostResponse, error) {
	p, err := s.repo.FindOwned(userID, id)
	if err != nil {
		return nil, err
	}
	p.IsPublished = !p.IsPublished
	p.PublishedAt = nil
	if p.IsPublished {
		t := s.now()
		p.PublishedAt = &t
	}
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

func (s *Service) Delete(userID, id uint) error { return s.repo.Delete(userID, id) }

func (s *Service) Stats(userID uint) (*Stats, error) {
	total, published, err := s.repo.Count(userID)
	if err != nil {
		return nil, err
	}
	return &Stats{
		Total:         total,
		TotalArticles: total,
		TotalTerbit:   published,
		TotalSelesai:  published,
		BelumSelesai:  total - published,
	}, nil
}