package auth

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"app/internal/models"
)

// Repository mengabstraksi akses data pengguna.
type Repository interface {
	Create(u *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByLogin(login string) (*models.User, error)
	ExistsOther(username, email string, excludeID uint) (bool, error)
	Save(u *models.User) error
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

func (r *GormRepository) Create(u *models.User) error { return r.db.Create(u).Error }

func (r *GormRepository) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *GormRepository) FindByLogin(login string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("username = ? OR email = ?", login, strings.ToLower(login)).First(&u).Error; err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *GormRepository) ExistsOther(username, email string, excludeID uint) (bool, error) {
	var count int64
	q := r.db.Model(&models.User{}).Where("id <> ?", excludeID).Where("username = ? OR email = ?", username, email)
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormRepository) Save(u *models.User) error { return r.db.Save(u).Error }
