package database

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/models"
)

var seq atomic.Int64

func memDB(t *testing.T) *gorm.DB {
	db, err := OpenMemory(fmt.Sprintf("dbtest%d", seq.Add(1)))
	require.NoError(t, err)
	return db
}

func closed(t *testing.T) *gorm.DB {
	db := memDB(t)
	sqlDB, _ := db.DB()
	require.NoError(t, sqlDB.Close())
	return db
}

func TestOpenMemoryAndMigrate(t *testing.T) {
	db := memDB(t)
	require.NoError(t, Migrate(db))
	assert.True(t, db.Migrator().HasTable(&models.User{}))
	assert.True(t, db.Migrator().HasTable(&models.Post{}))
	assert.Error(t, Migrate(closed(t)))
}

func TestConnectFailsWithoutServer(t *testing.T) {
	cfg := &config.Config{DBHost: "127.0.0.1", DBPort: "1", DBUsername: "x", DBPassword: "x", DBDatabase: "x", DBSSLMode: "disable"}
	_, err := Connect(cfg)
	assert.Error(t, err)

	cfgSQLite := &config.Config{DBConnection: "sqlite", DBDatabase: filepath.Join(t.TempDir(), "test_sqlite")}
	db, errSQLite := Connect(cfgSQLite)
	assert.NoError(t, errSQLite)
	assert.NotNil(t, db)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func TestSeed(t *testing.T) {
	db := memDB(t)
	require.NoError(t, Migrate(db))
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}
	require.NoError(t, Seed(db, cfg))
	require.NoError(t, Seed(db, cfg)) // idempoten

	var users []models.User
	require.NoError(t, db.Order("id").Find(&users).Error)
	require.Len(t, users, 3)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, models.RoleAdmin, users[0].Role)
	assert.Equal(t, "author", users[1].Username)
	assert.Equal(t, models.RoleAuthor, users[1].Role)
	assert.Equal(t, "member", users[2].Username)
	assert.Equal(t, models.RoleMember, users[2].Role)
	assert.NotEqual(t, "admin", users[0].Password, "password harus di-hash")
}

func TestSeedErrors(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}

	assert.Error(t, Seed(closed(t), cfg))

	db := memDB(t)
	require.NoError(t, Migrate(db))
	tooLong := make([]byte, 80)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	assert.Error(t, Seed(db, &config.Config{AdminUsername: "x", AdminPassword: string(tooLong)}))

	db2 := memDB(t)
	require.NoError(t, Migrate(db2))
	require.NoError(t, db2.Callback().Create().Before("gorm:create").Register("fail_user", func(tx *gorm.DB) {
		tx.AddError(errors.New("insert gagal"))
	}))
	assert.Error(t, Seed(db2, cfg))
}
