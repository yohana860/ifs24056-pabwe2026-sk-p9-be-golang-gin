package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const defaultAppKey = "delcomblog_secret_key_super_secure_2026_random_string_32chars"

// Config menampung seluruh konfigurasi aplikasi yang dibaca dari environment / .env.
type Config struct {
	AppEnv         string
	AppPort        string
	AppKey         string
	CORSOrigins    []string
	AdminUsername  string
	AdminPassword  string
	DBConnection   string
	DBHost         string
	DBPort         string
	DBDatabase     string
	DBUsername     string
	DBPassword     string
	DBSSLMode      string
	OpenAIEndpoint string
	OpenAIKey      string
	OpenAIModel    string
	SearchEndpoint string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// LoadConfig membaca berkas .env (jika ada) lalu membangun Config dengan nilai default.
func LoadConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		AppPort:        getEnv("APP_PORT", "8081"),
		AppKey:         getEnv("APP_KEY", defaultAppKey),
		CORSOrigins:    strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", "admin"),
		DBConnection:   getEnv("DB_CONNECTION", "pgsql"),
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBDatabase:     getEnv("DB_DATABASE", "pabwe_p9_blog"),
		DBUsername:     getEnv("DB_USERNAME", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		OpenAIEndpoint: getEnv("OPENAI_ENDPOINT", "https://delcom.org/api/v1"),
		OpenAIKey:      getEnv("OPENAI_KEY", ""),
		OpenAIModel:    getEnv("OPENAI_MODEL", ""),
		SearchEndpoint: getEnv("SEARCH_ENGINE_ENDPOINT", "https://delcom.org/api/v1/search"),
	}
}

// DSN membentuk connection string PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBUsername, c.DBPassword, c.DBDatabase, c.DBPort, c.DBSSLMode)
}

// Validate menolak konfigurasi tidak aman ketika dijalankan pada mode production.
func (c *Config) Validate() error {
	if c.AppEnv == "production" && (c.AppKey == defaultAppKey || len(c.AppKey) < 32) {
		return errors.New("APP_KEY wajib diganti dengan string acak minimal 32 karakter pada production")
	}
	return nil
}
