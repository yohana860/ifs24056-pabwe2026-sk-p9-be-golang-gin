// Package database mengelola koneksi, migrasi, dan seeder basis data.
package database

import (
	"os"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"app/internal/config"
	"app/internal/models"
)

// Open membuka koneksi GORM dari dialector apa pun.
func Open(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
}

// Connect membuka koneksi basis data untuk runtime aplikasi (PostgreSQL bila dikonfigurasi, atau SQLite).
func Connect(cfg *config.Config) (*gorm.DB, error) {
	if cfg.DBConnection == "sqlite" || (os.Getenv("DB_HOST") == "" && cfg.DBPort != "1") {
		return Open(sqlite.Open(cfg.DBDatabase + ".db"))
	}
	return Open(postgres.Open(cfg.DSN()))
}

// OpenMemory membuka SQLite in-memory (khusus unit test mandiri).
func OpenMemory(name string) (*gorm.DB, error) {
	db, err := Open(sqlite.Open("file:" + name + "?mode=memory&cache=shared"))
	if err == nil {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(1)
	}
	return db, err
}

// Migrate menjalankan AutoMigrate untuk seluruh entitas.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Post{})
}

// Seed menyuntikkan akun bawaan admin dan member bila belum ada.
func Seed(db *gorm.DB, cfg *config.Config) error {
	seeds := []struct{ username, password, role string }{
		{cfg.AdminUsername, cfg.AdminPassword, models.RoleAdmin},
		{"author", "author", models.RoleAuthor},
		{"member", "member", models.RoleMember},
	}
	for _, s := range seeds {
		var count int64
		if err := db.Model(&models.User{}).Where("username = ?", s.username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := models.User{Username: s.username, Email: s.username + "@example.com", Password: string(hash), Role: s.role}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}
	return nil
}
