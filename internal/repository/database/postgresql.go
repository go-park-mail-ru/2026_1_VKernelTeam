// Package postgresql реализует хранилище пользователей и объявлений на базе PostgreSQL.
// Используется пул соединений pgxpool для конкурентного доступа.
package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// Sentinel-ошибки — используются в usecase/auth для проверки через errors.Is.
var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

// GetAllAds возвращает список активных объявлений с агрегированными фото,
// количеством просмотров и избранного.
func (s *Storage) GetAllAds(ctx context.Context) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id,
			p.seller_id,
			p.category_id,
			p.title,
			p.description,
			p.price,
			p.status,
			p.created_at,
			p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			COUNT(DISTINCT pv.id)        AS views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count
		FROM product p
		LEFT JOIN product_image pi ON pi.product_id = p.id
		LEFT JOIN product_view  pv ON pv.product_id = p.id
		LEFT JOIN favorite       f ON f.product_id  = p.id
		WHERE p.deleted_at IS NULL
		  AND p.status = 'active'
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetAll: query: %w", err)
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID,
			&ad.SellerID,
			&ad.CategoryID,
			&ad.Title,
			&ad.Description,
			&ad.Price,
			&ad.Status,
			&ad.CreatedAt,
			&ad.UpdatedAt,
			&photos,
			&ad.ViewsCount,
			&ad.FavoritesCount,
		); err != nil {
			return nil, fmt.Errorf("GetAll: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetAll: rows: %w", err)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	return ads, nil
}

// Storage реализует интерфейсы auth.UserSaver, auth.UserProvider
// через пул соединений к PostgreSQL.
type Storage struct {
	pool *pgxpool.Pool
}

// New создаёт пул соединений и проверяет доступность БД.
// dsn — строка подключения, например:
//	"postgres://user:password@localhost:5432/clover?sslmode=disable"
func New(ctx context.Context, dsn string) (*Storage, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgresql.New: parse config: %w", err)
	}

	// Настройки пула (можно вынести в Config позже)
	cfg.MaxConns = 10
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgresql.New: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgresql.New: ping: %w", err)
	}

	return &Storage{pool: pool}, nil
}

// Close закрывает пул соединений. Вызывать при остановке приложения.
func (s *Storage) Close() {
	s.pool.Close()
}

// SaveUser сохраняет нового пользователя. Возвращает ErrUserExists,
// если email уже занят.
func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte, name string) (int64, error) {
	const query = `
		INSERT INTO "user" (first_name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := s.pool.QueryRow(ctx, query, name, email, passHash).Scan(&id)
	if err != nil {
		// pgx возвращает код 23505 при нарушении UNIQUE
		if isPgUniqueViolation(err) {
			return 0, ErrUserExists
		}
		return 0, fmt.Errorf("SaveUser: %w", err)
	}

	return id, nil
}

// User возвращает пользователя по email. Возвращает ErrUserNotFound,
// если пользователь не найден.
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
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
// В текущей схеме поля is_admin нет — всегда возвращает false.
func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
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
func (s *Storage) UserByID(ctx context.Context, userID int64) (models.User, error) {
	const query = `
		SELECT id, first_name, email, password_hash, created_at, updated_at
		FROM "user"
		WHERE id = $1`

	var u models.User
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Name, &u.Email, &u.PassHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("UserByID: %w", err)
	}

	return u, nil
}

// isPgUniqueViolation проверяет, является ли ошибка нарушением UNIQUE в PostgreSQL.
// Код 23505 — стандартный SQLSTATE для unique_violation.
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
