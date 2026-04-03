package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	postgres "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
}

func NewUserStorage(pool PgxIface) *UserStorage {
	return &UserStorage{pool: pool}
}

// SaveUser сохраняет нового пользователя. Возвращает ErrUserExists, если email уже занят.
func (s *UserStorage) SaveUser(ctx context.Context, email string, passHash []byte, name string) (int64, error) {
	const query = `
		INSERT INTO "user" (first_name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := s.pool.QueryRow(ctx, query, name, email, passHash).Scan(&id)
	if err != nil {
		if postgres.IsPgUniqueViolation(err) {
			return 0, ErrUserExists
		}
		return 0, fmt.Errorf("SaveUser: %w", err)
	}

	return id, nil
}

// User возвращает пользователя по email. Возвращает ErrUserNotFound, если он не найден.
func (s *UserStorage) User(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, first_name, email, password_hash, created_at, updated_at
		FROM "user"
		WHERE email = $1
	`

	var u models.User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("User: %w", err)
	}

	return u, nil
}

// IsAdmin проверяет, является ли пользователь с данным ID администратором.
func (s *UserStorage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const query = `SELECT id FROM "user" WHERE id = $1`

	var id int64
	err := s.pool.QueryRow(ctx, query, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrUserNotFound
		}
		return false, fmt.Errorf("IsAdmin: %w", err)
	}

	return false, nil
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
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("UserByID: %w", err)
	}

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

	var u models.User
	err := s.pool.QueryRow(ctx, query, name, userID).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("UpdateUser: %w", err)
	}

	return u, nil
}
