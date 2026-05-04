package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsPgUniqueViolation(t *testing.T) {
	t.Run("returns true on 23505", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23505"}
		assert.True(t, IsPgUniqueViolation(err))
	})

	t.Run("returns false on other pg code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23503"}
		assert.False(t, IsPgUniqueViolation(err))
	})

	t.Run("returns false on non-pg error", func(t *testing.T) {
		assert.False(t, IsPgUniqueViolation(errors.New("boom")))
	})

	t.Run("returns false on nil error", func(t *testing.T) {
		assert.False(t, IsPgUniqueViolation(nil))
	})
}

func TestNew_InvalidDSN(t *testing.T) {
	_, err := New("not a valid dsn://")
	assert.Error(t, err)
}

func TestNew_UnreachableDB(t *testing.T) {
	// валидная DSN, но порт заведомо закрыт — Ping упадёт.
	_, err := New("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
}
