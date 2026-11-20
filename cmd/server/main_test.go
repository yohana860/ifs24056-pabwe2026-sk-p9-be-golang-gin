package main

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/database"
)

func TestMainSuccessDoesNotExit(t *testing.T) {
	oldConnect, oldListen, oldExit := connect, listen, exit
	defer func() { connect, listen, exit = oldConnect, oldListen, oldExit }()
	t.Setenv("APP_ENV", "development")
	connect = func(*config.Config) (*gorm.DB, error) { return database.OpenMemory("maintest1") }
	listen = func(string, http.Handler) error { return nil }
	code := -1
	exit = func(c int) { code = c }
	main()
	assert.Equal(t, -1, code)
}

func TestMainExitsOnFailure(t *testing.T) {
	oldConnect, oldExit := connect, exit
	defer func() { connect, exit = oldConnect, oldExit }()
	connect = func(*config.Config) (*gorm.DB, error) { return nil, errors.New("db mati") }
	code := -1
	exit = func(c int) { code = c }
	main()
	assert.Equal(t, 1, code)
}

func TestListenAndServeReturnsError(t *testing.T) {
	assert.Error(t, listenAndServe("alamat-tidak-valid", http.NewServeMux()))
}
