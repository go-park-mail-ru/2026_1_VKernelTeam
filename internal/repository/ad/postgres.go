package ad

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PgxPool интерфейс для пула соединений (или транзакции),
// позволяющий подменять его моком в тестах.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// Sentinel-ошибки
var (
	ErrAdNotFound  = errors.New("ad not found")
	ErrAdForbidden = errors.New("forbidden: not the owner")
)

// AdStorage отвечает за операции с объявлениями.
type AdStorage struct {
	pool PgxPool
}

func NewAdStorage(pool PgxPool) *AdStorage {
	return &AdStorage{pool: pool}
}

// GetAdByID возвращает объявление по ID. Возвращает ErrAdNotFound, если оно не найдено.
func (s *AdStorage) GetAdByID(ctx context.Context, id int64) (models.Ad, error) {
	const query = `
		SELECT
			p.id,
			p.seller_id,
			p.category_id,
			p.title,
			p.description,
			p.price,
			p.status,
			p.location,
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
		WHERE p.id = $1
		  AND p.deleted_at IS NULL
		GROUP BY p.id
	`

	var ad models.Ad
	var photos []string
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&ad.ID,
		&ad.SellerID,
		&ad.CategoryID,
		&ad.Title,
		&ad.Description,
		&ad.Price,
		&ad.Status,
		&ad.Location,
		&ad.CreatedAt,
		&ad.UpdatedAt,
		&photos,
		&ad.ViewsCount,
		&ad.FavoritesCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ad{}, ErrAdNotFound
		}
		return models.Ad{}, fmt.Errorf("GetAdByID: %w", err)
	}
	ad.Photos = photos
	return ad, nil
}

// GetAllAds возвращает список активных объявлений.
func (s *AdStorage) GetAllAds(ctx context.Context) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id,
			p.seller_id,
			p.category_id,
			p.title,
			p.description,
			p.price,
			p.status,
			COALESCE(p.location, '') AS location,
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
		return nil, fmt.Errorf("GetAllAds: query: %w", err)
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
			&ad.Location,
			&ad.CreatedAt,
			&ad.UpdatedAt,
			&photos,
			&ad.ViewsCount,
			&ad.FavoritesCount,
		); err != nil {
			return nil, fmt.Errorf("GetAllAds: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetAllAds: rows: %w", err)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	return ads, nil
}

// CreateAd создает новое объявление и возвращает его ID.
func (s *AdStorage) CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error) {
	const query = `
		INSERT INTO product (seller_id, category_id, title, description, price, status, location)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var adID int64
	err := s.pool.QueryRow(ctx, query,
		req.UserID,
		req.CategoryID,
		req.Title,
		req.Description,
		req.Price,
		req.Status,
		req.Location,
	).Scan(&adID)
	if err != nil {
		return 0, fmt.Errorf("CreateAd: insert: %w", err)
	}

	return adID, nil
}

// AddProductImages добавляет изображения к объявлению в таблицу product_image.
func (s *AdStorage) AddProductImages(ctx context.Context, adID int64, photos []string) error {
	if len(photos) == 0 {
		return nil
	}

	const baseQuery = `INSERT INTO product_image (product_id, file_path, sort_order) VALUES `

	args := make([]any, 0, len(photos)*3)
	var sb strings.Builder
	for i, photo := range photos {
		if i > 0 {
			sb.WriteString(", ")
		}
		paramIdx := i * 3
		fmt.Fprintf(&sb, "($%d, $%d, $%d)", paramIdx+1, paramIdx+2, paramIdx+3)
		args = append(args, adID, photo, i)
	}

	_, err := s.pool.Exec(ctx, baseQuery+sb.String(), args...)
	if err != nil {
		return fmt.Errorf("AddProductImages: exec: %w", err)
	}

	return nil
}

// UpdateAd обновляет объявление. Проверяет принадлежность объявления пользователю.
func (s *AdStorage) UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error {
	const query = `
		UPDATE product
		SET category_id = $1,
		    title       = $2,
		    description = $3,
		    price       = $4,
		    status      = $5,
		    location    = $6,
		    updated_at  = NOW()
		WHERE id = $7
		  AND seller_id = $8
		  AND deleted_at IS NULL
	`

	result, err := s.pool.Exec(ctx, query,
		req.CategoryID,
		req.Title,
		req.Description,
		req.Price,
		req.Status,
		req.Location,
		req.ID,
		req.UserID,
	)
	if err != nil {
		return fmt.Errorf("UpdateAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAdNotFound
	}

	return nil
}

// DeleteAd выполняет мягкое удаление объявления (устанавливает deleted_at).
func (s *AdStorage) DeleteAd(ctx context.Context, id int64, userID int64) error {
	const query = `
		UPDATE product
		SET deleted_at = NOW()
		WHERE id = $1
		  AND seller_id = $2
		  AND deleted_at IS NULL
	`

	result, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("DeleteAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAdNotFound
	}

	return nil
}

// CloseAd закрывает объявление (устанавливает статус 'archived').
func (s *AdStorage) CloseAd(ctx context.Context, id int64, userID int64) error {
	const query = `
		UPDATE product
		SET status     = 'archived',
		    updated_at = NOW()
		WHERE id = $1
		  AND seller_id = $2
		  AND deleted_at IS NULL
		  AND status != 'archived'
	`

	result, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("CloseAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAdNotFound
	}

	return nil
}

// GetAdsByUserID возвращает список всех объявлений пользователя по его ID.
func (s *AdStorage) GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id, p.seller_id, p.category_id, p.title, p.description,
			p.price, p.status, p.created_at, p.updated_at,
			COALESCE(array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL), '{}') AS photos,
			COUNT(DISTINCT pv.id) AS views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count
		FROM product p
		LEFT JOIN product_image pi ON pi.product_id = p.id
		LEFT JOIN product_view pv ON pv.product_id = p.id
		LEFT JOIN favorite f ON f.product_id = p.id
		WHERE p.seller_id = $1 AND p.deleted_at IS NULL
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("GetAdsByUserID: query: %w", err)
	}

	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID, &ad.SellerID, &ad.CategoryID, &ad.Title, &ad.Description,
			&ad.Price, &ad.Status, &ad.CreatedAt, &ad.UpdatedAt,
			&photos, &ad.ViewsCount, &ad.FavoritesCount,
		); err != nil {
			return nil, fmt.Errorf("GetAdsByUserID: scan: %w", err)
		}

		ad.Photos = photos
		ads = append(ads, ad)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	return ads, nil
}
