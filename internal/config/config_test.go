package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func clearEnv(t *testing.T) {
	for _, k := range []string{"APP_ENV", "APP_PORT", "APP_KEY", "CORS_ALLOWED_ORIGINS", "ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
		"OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL", "SEARCH_ENGINE_ENDPOINT"} {
		t.Setenv(k, "")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	clearEnv(t)
	cfg := LoadConfig()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, defaultAppKey, cfg.AppKey)
	assert.Equal(t, []string{"*"}, cfg.CORSOrigins)
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "https://delcom.org/api/v1", cfg.OpenAIEndpoint)
	assert.Empty(t, cfg.OpenAIKey)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEndpoint)
}

func TestLoadConfigFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_PORT", "9999")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.test,http://b.test")
	t.Setenv("OPENAI_KEY", "secret")
	cfg := LoadConfig()
	assert.Equal(t, "9999", cfg.AppPort)
	assert.Equal(t, []string{"http://a.test", "http://b.test"}, cfg.CORSOrigins)
	assert.Equal(t, "secret", cfg.OpenAIKey)
}

func TestDSN(t *testing.T) {
	cfg := &Config{DBHost: "h", DBUsername: "u", DBPassword: "p", DBDatabase: "d", DBPort: "1", DBSSLMode: "disable"}
	assert.Equal(t, "host=h user=u password=p dbname=d port=1 sslmode=disable TimeZone=UTC", cfg.DSN())
}

func TestValidate(t *testing.T) {
	assert.NoError(t, (&Config{AppEnv: "development", AppKey: defaultAppKey}).Validate())
	assert.Error(t, (&Config{AppEnv: "production", AppKey: defaultAppKey}).Validate())
	assert.Error(t, (&Config{AppEnv: "production", AppKey: "short"}).Validate())
	assert.NoError(t, (&Config{AppEnv: "production", AppKey: "0123456789abcdef0123456789abcdef-x"}).Validate())
}
