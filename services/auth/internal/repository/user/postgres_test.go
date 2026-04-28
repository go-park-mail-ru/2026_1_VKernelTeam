package user_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/user"
)

func TestUserStorage_SaveUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := user.NewUserStorage(mock, slog.Default())
	ctx := context.Background()

	email := "test@mail.ru"
	pass := []byte("hash")
	name := "Ivan"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO "user"`).
			WithArgs(name, email, pass).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))

		id, err := repo.SaveUser(ctx, email, pass, name)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("user_exists_violation", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23505"}
		mock.ExpectQuery(`INSERT INTO "user"`).
			WithArgs(name, email, pass).
			WillReturnError(pgErr)

		id, err := repo.SaveUser(ctx, email, pass, name)
		assert.ErrorIs(t, err, user.ErrUserExists)
		assert.Equal(t, int64(0), id)
	})
}

func TestUserStorage_User(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	repo := user.NewUserStorage(mock, slog.Default())
	ctx := context.Background()
	email := "test@mail.ru"

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "first_name", "email", "password_hash", "role", "created_at", "updated_at"}).
			AddRow(int64(1), "Ivan", email, []byte("hash"), "user", now, now)

		mock.ExpectQuery(`SELECT id, first_name, email`).
			WithArgs(email).
			WillReturnRows(rows)

		u, err := repo.User(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, email, u.Email)
		assert.Equal(t, "Ivan", u.Name)
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, first_name, email`).
			WithArgs(email).
			WillReturnError(pgx.ErrNoRows)

		u, err := repo.User(ctx, email)
		assert.ErrorIs(t, err, user.ErrUserNotFound)
		assert.Empty(t, u.Email)
	})
}

func TestUserStorage_UserByID(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	repo := user.NewUserStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		columns := []string{
			"id", "first_name", "email", "password_hash", "avatar_path",
			"rating", "role", "created_at", "updated_at",
		}

		rows := pgxmock.NewRows(columns).
			AddRow(userID, "Ivan", "test@mail.ru", []byte("hash"), "/img/ava.png",
				4.5, "user", now, now)

		mock.ExpectQuery(`SELECT`).
			WithArgs(userID).
			WillReturnRows(rows)

		u, err := repo.UserByID(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, userID, u.ID)
		assert.Equal(t, "/img/ava.png", u.AvatarPath)
	})
}

func TestUserStorage_UpdateUser(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	repo := user.NewUserStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("success_update", func(t *testing.T) {
		newName := "NewName"
		userID := int64(1)

		mock.ExpectQuery(`UPDATE "user"`).
			WithArgs(newName, userID).
			WillReturnRows(pgxmock.NewRows([]string{"id", "first_name", "email", "password_hash", "role", "created_at", "updated_at"}).
				AddRow(userID, newName, "test@mail.ru", []byte("hash"), "user", time.Now(), time.Now()))

		u, err := repo.UpdateUser(ctx, userID, newName)
		assert.NoError(t, err)
		assert.Equal(t, newName, u.Name)
	})
}

func TestUserStorage_UpdateAvatarPath(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	repo := user.NewUserStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)
	newPath := "/static/img/avatars/1_12345.png"

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE "user" SET avatar_path = \$1`).
			WithArgs(newPath, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateAvatarPath(ctx, userID, newPath)
		assert.NoError(t, err)
	})

	t.Run("user_not_found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE "user" SET avatar_path = \$1`).
			WithArgs(newPath, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateAvatarPath(ctx, userID, newPath)
		assert.Error(t, err)
	})
}
