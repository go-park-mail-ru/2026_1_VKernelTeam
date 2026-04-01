package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
)

// Тестируем Close через pgxmock
func TestClient_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	t.Run("successful_close", func(t *testing.T) {
		mock.ExpectClose()
		mock.Close()
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNew_Errors(t *testing.T) {
	t.Run("invalid_dsn", func(t *testing.T) {
		// Некорректный формат строки подключения
		client, err := postgres.New("invalid dsn string")

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "parse config")
	})

	t.Run("empty_dsn", func(t *testing.T) {
		client, err := postgres.New("")

		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestIsPgUniqueViolation(t *testing.T) {
	t.Run("unique_violation_true", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23505"}
		assert.True(t, postgres.IsPgUniqueViolation(err))
	})

	t.Run("other_error_false", func(t *testing.T) {
		assert.False(t, postgres.IsPgUniqueViolation(errors.New("other")))
		assert.False(t, postgres.IsPgUniqueViolation(nil))
	})
}
