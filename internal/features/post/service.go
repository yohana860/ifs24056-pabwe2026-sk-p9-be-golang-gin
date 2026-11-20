package post

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"app/internal/models"
)

// ErrInvalidCover dikembalikan bila cover_image_path bukan URL http(s) atau path relatif aman.
var ErrInvalidCover = errors.New("cover_image_path harus berupa URL http(s) atau path yang diawali /")

// Service berisi logika bisnis artikel.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService membuat Service artikel.
func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

// Categories mengembalikan daftar kategori sah.
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
		ID: p.ID, Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		CoverImagePath: p.CoverImagePath, IsPublished: p.IsPublished, PublishedAt: p.PublishedAt,
		ReadingTime: readingTime(p.Content), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if withContent {
		res.Content = p.Content
	}
	if p.User != nil {
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

// ListPublic mengembalikan artikel terbit (tanpa isi lengkap) sesuai filter.
func (s *Service) ListPublic(f Filter) ([]PostResponse, error) {
	f.Q = strings.TrimSpace(f.Q)
	if f.Category == "Semua" {
		f.Category = ""
	}
	posts, err := s.repo.ListPublished(f)
	if err != nil {
		return nil, err
	}
	return toList(posts), nil
}

// GetPublic mengambil detail artikel terbit berdasarkan slug. Draf tidak pernah dikembalikan.
func (s *Service) GetPublic(slug string) (*PostResponse, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	if err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

// ListMine mengembalikan seluruh artikel milik penulis (terbit dan draf).
func (s *Service) ListMine(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	return toList(posts), nil
}

// GetMine mengambil satu artikel milik penulis untuk diedit.
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
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// Create membuat artikel draf baru dengan slug unik.
func (s *Service) Create(userID uint, req PostRequest) (*PostResponse, error) {
	if !validCover(req.CoverImagePath) {
		return nil, ErrInvalidCover
	}
	slug, err := s.uniqueSlug(req.Title)
	if err != nil {
		return nil, err
	}
	p := &models.Post{
		UserID: userID, Title: strings.TrimSpace(req.Title), Slug: slug, Category: req.Category,
		Excerpt: req.Excerpt, Content: req.Content, CoverImagePath: req.CoverImagePath,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

// Update memperbarui artikel milik penulis; slug tetap agar tautan lama tidak rusak.
func (s *Service) Update(userID, id uint, req PostRequest) (*PostResponse, error) {
	if !validCover(req.CoverImagePath) {
		return nil, ErrInvalidCover
	}
	p, err := s.repo.FindOwned(userID, id)
	if err != nil {
		return nil, err
	}
	p.Title, p.Category, p.Excerpt = strings.TrimSpace(req.Title), req.Category, req.Excerpt
	p.Content, p.CoverImagePath = req.Content, req.CoverImagePath
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	res := toResponse(p, true)
	return &res, nil
}

// TogglePublish membalik status Draf <-> Terbit beserta published_at.
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

// Delete menghapus artikel milik penulis.
func (s *Service) Delete(userID, id uint) error { return s.repo.Delete(userID, id) }

// Stats menghitung statistik artikel penulis langsung dari database.
func (s *Service) Stats(userID uint) (*Stats, error) {
	total, published, err := s.repo.Count(userID)
	if err != nil {
		return nil, err
	}
	return &Stats{Total: total, TotalTerbit: published, BelumSelesai: total - published}, nil
}
