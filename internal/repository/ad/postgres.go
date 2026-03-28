package ad

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdStorage отвечает за операции с объявлениями.
type AdStorage struct {
	pool *pgxpool.Pool
}

func NewAdStorage(pool *pgxpool.Pool) *AdStorage {
	return &AdStorage{pool: pool}
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
