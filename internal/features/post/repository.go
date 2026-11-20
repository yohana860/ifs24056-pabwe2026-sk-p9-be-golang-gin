package post

import (
	"errors"

	"gorm.io/gorm"

	"app/internal/models"
)

// ErrNotFound dikembalikan ketika artikel tidak ada atau bukan milik pengguna.
var ErrNotFound = errors.New("artikel tidak ditemukan")

// Repository mengabstraksi akses data artikel.
type Repository interface {
	ListPublished(f Filter) ([]models.Post, error)
	FindPublishedBySlug(slug string) (*models.Post, error)
	ListByUser(userID uint) ([]models.Post, error)
	FindOwned(userID, id uint) (*models.Post, error)
	Create(p *models.Post) error
	Save(p *models.Post) error
	Delete(userID, id uint) error
	SlugExists(slug string) (bool, error)
	Count(userID uint) (total, published int64, err error)
}

// GormRepository adalah implementasi Repository berbasis GORM.
type GormRepository struct{ db *gorm.DB }

// NewRepository membuat GormRepository.
func NewRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func authorOnly(db *gorm.DB) *gorm.DB { return db.Select("id", "username") }

func (r *GormRepository) ListPublished(f Filter) ([]models.Post, error) {
	q := r.db.Preload("User", authorOnly).Where("is_published = ?", true)
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Q != "" {
		like := "%" + escapeLike(f.Q) + "%"
		q = q.Where("(LOWER(title) LIKE LOWER(?) ESCAPE '\\' OR LOWER(excerpt) LIKE LOWER(?) ESCAPE '\\' OR LOWER(content) LIKE LOWER(?) ESCAPE '\\')", like, like, like)
	}
	posts := []models.Post{}
	if err := q.Order("published_at DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *GormRepository) FindPublishedBySlug(slug string) (*models.Post, error) {
	var p models.Post
	err := r.db.Preload("User", authorOnly).Where("slug = ? AND is_published = ?", slug, true).First(&p).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

func (r *GormRepository) ListByUser(userID uint) ([]models.Post, error) {
	posts := []models.Post{}
	if err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *GormRepository) FindOwned(userID, id uint) (*models.Post, error) {
	var p models.Post
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&p).Error; err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

func (r *GormRepository) Create(p *models.Post) error { return r.db.Create(p).Error }

func (r *GormRepository) Save(p *models.Post) error { return r.db.Save(p).Error }

func (r *GormRepository) Delete(userID, id uint) error {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Post{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormRepository) SlugExists(slug string) (bool, error) {
	var n int64
	if err := r.db.Model(&models.Post{}).Where("slug = ?", slug).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *GormRepository) Count(userID uint) (int64, int64, error) {
	var total, published int64
	base := r.db.Model(&models.Post{}).Where("user_id = ?", userID)
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := base.Where("is_published = ?", true).Count(&published).Error; err != nil {
		return 0, 0, err
	}
	return total, published, nil
}

func escapeLike(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\\' || r == '%' || r == '_' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}
