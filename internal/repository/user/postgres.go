package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	postgres "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opSaveUser         = "db.user.SaveUser"
	opGetUser          = "db.user.User"
	opGetUserByID      = "db.user.UserByID"
	opUpdateUser       = "db.user.UpdateUser"
	opUpdateAvatarPath = "db.user.UpdateAvatarPath"
	opGetUserRole      = "db.user.GetUserRole"
)

// Sentinel-ошибки — используются в юзкейсе для проверки через errors.Is.
var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

type PgxIface interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// UserStorage отвечает за операции с пользователями.
type UserStorage struct {
	pool PgxIface
	log  *slog.Logger
}

func NewUserStorage(pool PgxIface, log *slog.Logger) *UserStorage {
	return &UserStorage{pool: pool, log: log}
}

// SaveUser сохраняет нового пользователя. Возвращает ErrUserExists, если email уже занят.
func (s *UserStorage) SaveUser(ctx context.Context, email string, passHash []byte, name string) (int64, error) {
	const query = `
		INSERT INTO "user" (first_name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opSaveUser),
		slog.String("email", email),
	)

	var id int64
	err := s.pool.QueryRow(ctx, query, name, email, passHash).Scan(&id)
	if err != nil {
		if postgres.IsPgUniqueViolation(err) {
			s.log.WarnContext(ctx, "user already exists",
				slog.String("op", opSaveUser),
				slog.String("email", email),
			)
			return 0, ErrUserExists
		}
		s.log.ErrorContext(ctx, "failed to save user",
			slog.String("op", opSaveUser),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("SaveUser: %w", err)
	}

	s.log.InfoContext(ctx, "user saved successfully",
		slog.String("op", opSaveUser),
		slog.Int64("id", id),
	)
	return id, nil
}

// User возвращает пользователя по email. Возвращает ErrUserNotFound, если он не найден.
func (s *UserStorage) User(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, first_name, email, password_hash, created_at, updated_at
		FROM "user"
		WHERE email = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetUser),
		slog.String("email", email),
	)

	var u models.User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.DebugContext(ctx, "user not found",
				slog.String("op", opGetUser),
				slog.String("email", email),
			)
			return models.User{}, ErrUserNotFound
		}
		s.log.ErrorContext(ctx, "failed to get user",
			slog.String("op", opGetUser),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("User: %w", err)
	}

	s.log.DebugContext(ctx, "user fetched successfully",
		slog.String("op", opGetUser),
		slog.Int64("id", u.ID),
	)
	return u, nil
}

// UserByID возвращает пользователя по его ID.
func (s *UserStorage) UserByID(ctx context.Context, userID int64) (models.User, error) {
	const query = `
		SELECT
            u.id, u.first_name, u.email, u.password_hash,
            COALESCE(u.avatar_path, '') as avatar_path,
            u.rating, u.created_at, u.updated_at,
            (SELECT COUNT(*) FROM review WHERE receiver_id = u.id) as reviews_count,
            (SELECT COUNT(*) FROM product WHERE seller_id = u.id AND deleted_at IS NULL) as ads_count,
            (SELECT COUNT(*) FROM favorite WHERE user_id = u.id) as favorites_count,
            (SELECT SUM(quantity) FROM cart_item WHERE user_id = u.id) as cart_count,
            (SELECT COUNT(*) FROM message WHERE chat_id IN (
                SELECT id FROM chat WHERE buyer_id = u.id OR seller_id = u.id
            ) AND sender_id != u.id AND is_read = false) as unread_count
        FROM "user" u
        WHERE u.id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetUserByID),
		slog.Int64("user_id", userID),
	)

	var u models.User
	var cartCount *int

	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.AvatarPath, &u.Rating, &u.CreatedAt, &u.UpdatedAt,
		&u.ReviewsCount, &u.AdsCount, &u.FavoritesCount,
		&cartCount, &u.MessagesCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.DebugContext(ctx, "user not found",
				slog.String("op", opGetUserByID),
				slog.Int64("user_id", userID),
			)
			return models.User{}, ErrUserNotFound
		}
		s.log.ErrorContext(ctx, "failed to get user by id",
			slog.String("op", opGetUserByID),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("UserByID: %w", err)
	}

	s.log.DebugContext(ctx, "user fetched by id successfully",
		slog.String("op", opGetUserByID),
		slog.Int64("user_id", userID),
	)
	return u, nil
}

// UpdateUser обновляет данные пользователя в БД и возвращает обновленную модель.
func (s *UserStorage) UpdateUser(ctx context.Context, userID int64, name string) (models.User, error) {
	const query = `
		UPDATE "user"
		SET first_name = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id, first_name, email, password_hash, created_at, updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opUpdateUser),
		slog.Int64("user_id", userID),
	)

	var u models.User
	err := s.pool.QueryRow(ctx, query, name, userID).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.WarnContext(ctx, "user not found for update",
				slog.String("op", opUpdateUser),
				slog.Int64("user_id", userID),
			)
			return models.User{}, ErrUserNotFound
		}
		s.log.ErrorContext(ctx, "failed to update user",
			slog.String("op", opUpdateUser),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("UpdateUser: %w", err)
	}

	s.log.InfoContext(ctx, "user updated successfully",
		slog.String("op", opUpdateUser),
		slog.Int64("user_id", userID),
	)
	return u, nil
}

// GetUserRole возвращает роль пользователя по его ID.
func (s *UserStorage) GetUserRole(ctx context.Context, userID int64) (string, error) {
	const query = `SELECT role FROM "user" WHERE id = $1`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetUserRole),
		slog.Int64("user_id", userID),
	)

	var role string
	err := s.pool.QueryRow(ctx, query, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotFound
		}
		s.log.ErrorContext(ctx, "failed to get user role",
			slog.String("op", opGetUserRole),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("GetUserRole: %w", err)
	}

	return role, nil
}

// UpdateAvatarPath обновляет путь к аватару пользователя
func (s *UserStorage) UpdateAvatarPath(ctx context.Context, userID int64, path string) error {
	const query = `UPDATE "user" SET avatar_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opUpdateAvatarPath),
		slog.Int64("user_id", userID),
	)

	res, err := s.pool.Exec(ctx, query, path, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to update avatar path",
			slog.String("op", opUpdateAvatarPath),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("UpdateAvatarPath: %w", err)
	}

	if res.RowsAffected() == 0 {
		s.log.WarnContext(ctx, "user not found for avatar update",
			slog.String("op", opUpdateAvatarPath),
			slog.Int64("user_id", userID),
		)
		return ErrUserNotFound
	}

	s.log.InfoContext(ctx, "avatar path updated successfully",
		slog.String("op", opUpdateAvatarPath),
		slog.Int64("user_id", userID),
	)
	return nil
}
