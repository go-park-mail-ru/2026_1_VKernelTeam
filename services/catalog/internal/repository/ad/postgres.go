package ad

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opGetAdByID                       = "db.ad.GetAdByID"
	opGetAllAds                       = "db.ad.GetAllAds"
	opSearchAds                       = "db.ad.SearchAds"
	opCreateAd                        = "db.ad.CreateAd"
	opAddProductImages                = "db.ad.AddProductImages"
	opDeleteProductImages             = "db.ad.DeleteProductImages"
	opUpdateAd                        = "db.ad.UpdateAd"
	opDeleteAd                        = "db.ad.DeleteAd"
	opCloseAd                         = "db.ad.CloseAd"
	opGetAdsByUserID                  = "db.ad.GetAdsByUserID"
	opAddFavorite                     = "db.ad.AddFavorite"
	opRemoveFavorite                  = "db.ad.RemoveFavorite"
	opGetUserFavorites                = "db.ad.GetUserFavorites"
	opGetProductCharacteristics       = "db.ad.getProductCharacteristics"
	opGetProductCustomCharacteristics = "db.ad.getProductCustomCharacteristics"
	opSetProductCharacteristics       = "db.ad.SetProductCharacteristics"
	opSetProductCustomCharacteristics = "db.ad.SetProductCustomCharacteristics"
	opGetCategoryCharacteristics      = "db.ad.GetCategoryCharacteristics"
	opGetPriceHistory                 = "db.ad.GetPriceHistory"
	opAdminDeleteAd                   = "db.ad.AdminDeleteAd"
	opModerateAd                      = "db.ad.ModerateAd"
	opGetModerationQueue              = "db.ad.GetModerationQueue"
	opGetUserAdsByStatus              = "db.ad.GetUserAdsByStatus"
)

// PgxPool интерфейс для пула соединений (или транзакции),
// позволяющий подменять его моком в тестах.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Sentinel-ошибки
var (
	ErrAdNotFound  = errors.New("ad not found")
	ErrAdForbidden = errors.New("forbidden: not the owner")
)

// AdStorage отвечает за операции с объявлениями.
type AdStorage struct {
	pool PgxPool
	log  *slog.Logger
}

// NewAdStorage создаёт хранилище объявлений на базе PostgreSQL.
func NewAdStorage(pool PgxPool, log *slog.Logger) *AdStorage {
	return &AdStorage{pool: pool, log: log}
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
			p.lat,
			p.lon,
			p.created_at,
			p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			p.views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count,
			COALESCE(vpp.is_boosted, false)     AS is_boosted,
			COALESCE(vpp.is_highlighted, false) AS is_highlighted
		FROM product p
		LEFT JOIN product_image       pi  ON pi.product_id  = p.id
		LEFT JOIN favorite            f   ON f.product_id   = p.id
		LEFT JOIN v_product_promotion vpp ON vpp.product_id = p.id
		WHERE p.id = $1
		  AND p.deleted_at IS NULL
		  AND p.status <> 'admin_deleted'
		GROUP BY p.id, vpp.is_boosted, vpp.is_highlighted
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetAdByID),
		slog.Int64("ad_id", id),
	)

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
		&ad.Lat,
		&ad.Lon,
		&ad.CreatedAt,
		&ad.UpdatedAt,
		&photos,
		&ad.ViewsCount,
		&ad.FavoritesCount,
		&ad.IsBoosted,
		&ad.IsHighlighted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.DebugContext(ctx, "ad not found",
				slog.String("op", opGetAdByID),
				slog.Int64("ad_id", id),
			)
			return models.Ad{}, ErrAdNotFound
		}
		s.log.ErrorContext(ctx, "failed to get ad by id",
			slog.String("op", opGetAdByID),
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		return models.Ad{}, fmt.Errorf("GetAdByID: %w", err)
	}
	ad.Photos = photos

	catChars, err := s.getProductCharacteristics(ctx, []int64{id})
	if err != nil {
		return models.Ad{}, fmt.Errorf("GetAdByID: %w", err)
	}
	ad.CategoryCharacteristics = catChars[id]
	if ad.CategoryCharacteristics == nil {
		ad.CategoryCharacteristics = []models.ProductCharacteristic{}
	}

	customChars, err := s.getProductCustomCharacteristics(ctx, []int64{id})
	if err != nil {
		return models.Ad{}, fmt.Errorf("GetAdByID: %w", err)
	}
	ad.CustomCharacteristics = customChars[id]
	if ad.CustomCharacteristics == nil {
		ad.CustomCharacteristics = []models.ProductCustomCharacteristic{}
	}

	s.log.DebugContext(ctx, "ad fetched successfully",
		slog.String("op", opGetAdByID),
		slog.Int64("ad_id", id),
	)
	return ad, nil
}

// GetAllAds возвращает список активных объявлений. Сортировка: забустенные сверху, затем по дате.
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
			p.lat,
			p.lon,
			p.created_at,
			p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			p.views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count,
			COALESCE(vpp.is_boosted, false)     AS is_boosted,
			COALESCE(vpp.is_highlighted, false) AS is_highlighted
		FROM product p
		LEFT JOIN product_image       pi  ON pi.product_id  = p.id
		LEFT JOIN favorite            f   ON f.product_id   = p.id
		LEFT JOIN v_product_promotion vpp ON vpp.product_id = p.id
		WHERE p.deleted_at IS NULL
		  AND p.status = 'active'
		GROUP BY p.id, vpp.is_boosted, vpp.is_highlighted
		ORDER BY
			COALESCE(vpp.is_boosted, false) DESC,
			p.created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetAllAds),
	)

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query all ads",
			slog.String("op", opGetAllAds),
			slog.String("error", err.Error()),
		)
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
			&ad.Lat,
			&ad.Lon,
			&ad.CreatedAt,
			&ad.UpdatedAt,
			&photos,
			&ad.ViewsCount,
			&ad.FavoritesCount,
			&ad.IsBoosted,
			&ad.IsHighlighted,
		); err != nil {
			s.log.ErrorContext(ctx, "failed to scan ad row",
				slog.String("op", opGetAllAds),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("GetAllAds: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		s.log.ErrorContext(ctx, "rows iteration error",
			slog.String("op", opGetAllAds),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetAllAds: rows: %w", err)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetAllAds: load characteristics: %w", err)
	}

	s.log.DebugContext(ctx, "all ads fetched successfully",
		slog.String("op", opGetAllAds),
		slog.Int("count", len(ads)),
	)
	return ads, nil
}

// CreateAd создает новое объявление и возвращает его ID.
func (s *AdStorage) CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error) {
	const query = `
		INSERT INTO product (seller_id, category_id, title, description, price, status, location, lat, lon)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCreateAd),
		slog.Int64("user_id", req.UserID),
		slog.String("title", req.Title),
	)

	var adID int64
	err := s.pool.QueryRow(ctx, query,
		req.UserID,
		req.CategoryID,
		req.Title,
		req.Description,
		req.Price,
		req.Status,
		req.Location,
		req.Lat,
		req.Lon,
	).Scan(&adID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create ad",
			slog.String("op", opCreateAd),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("CreateAd: insert: %w", err)
	}

	s.log.InfoContext(ctx, "ad created in database",
		slog.String("op", opCreateAd),
		slog.Int64("ad_id", adID),
	)
	return adID, nil
}

// AddProductImages добавляет изображения к объявлению в таблицу product_image.
func (s *AdStorage) AddProductImages(ctx context.Context, adID int64, photos []string) error {
	if len(photos) == 0 {
		return nil
	}

	const baseQuery = `INSERT INTO product_image (product_id, file_path, sort_order) VALUES `

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opAddProductImages),
		slog.Int64("ad_id", adID),
		slog.Int("photos_count", len(photos)),
	)

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
		s.log.ErrorContext(ctx, "failed to add product images",
			slog.String("op", opAddProductImages),
			slog.Int64("ad_id", adID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("AddProductImages: exec: %w", err)
	}

	s.log.DebugContext(ctx, "product images added",
		slog.String("op", opAddProductImages),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// DeleteProductImages удаляет все изображения объявления.
func (s *AdStorage) DeleteProductImages(ctx context.Context, adID int64) error {
	const query = `DELETE FROM product_image WHERE product_id = $1`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opDeleteProductImages),
		slog.Int64("ad_id", adID),
	)

	_, err := s.pool.Exec(ctx, query, adID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to delete product images",
			slog.String("op", opDeleteProductImages),
			slog.Int64("ad_id", adID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("DeleteProductImages: exec: %w", err)
	}

	s.log.DebugContext(ctx, "product images deleted",
		slog.String("op", opDeleteProductImages),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// UpdateAd обновляет объявление. Проверяет принадлежность объявления пользователю.
// Обновляет только те поля, которые были отправлены (не nil).
func (s *AdStorage) UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error {
	var updates []string
	var args []any

	argNum := 1

	if req.CategoryID != nil {
		updates = append(updates, fmt.Sprintf("category_id = $%d", argNum))
		args = append(args, *req.CategoryID)
		argNum++
	}
	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *req.Title)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.Price != nil {
		updates = append(updates, fmt.Sprintf("price = $%d", argNum))
		args = append(args, *req.Price)
		argNum++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argNum))
		args = append(args, *req.Location)
		argNum++
	}
	if req.Lat != nil {
		updates = append(updates, fmt.Sprintf("lat = $%d", argNum))
		args = append(args, *req.Lat)
		argNum++
	}
	if req.Lon != nil {
		updates = append(updates, fmt.Sprintf("lon = $%d", argNum))
		args = append(args, *req.Lon)
		argNum++
	}

	if len(updates) == 0 {
		return fmt.Errorf("UpdateAd: no fields to update")
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE product
		SET %s
		WHERE id = $%d
		  AND seller_id = $%d
		  AND deleted_at IS NULL
	`, strings.Join(updates, ", "), argNum, argNum+1)

	args = append(args, req.ID, req.UserID)

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opUpdateAd),
		slog.Int64("ad_id", req.ID),
		slog.Int64("user_id", req.UserID),
	)

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to update ad",
			slog.String("op", opUpdateAd),
			slog.Int64("ad_id", req.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("UpdateAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		s.log.WarnContext(ctx, "ad not found or not owned by user",
			slog.String("op", opUpdateAd),
			slog.Int64("ad_id", req.ID),
			slog.Int64("user_id", req.UserID),
		)
		return ErrAdNotFound
	}

	s.log.InfoContext(ctx, "ad updated in database",
		slog.String("op", opUpdateAd),
		slog.Int64("ad_id", req.ID),
	)
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

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opDeleteAd),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)

	result, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to delete ad",
			slog.String("op", opDeleteAd),
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("DeleteAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		s.log.WarnContext(ctx, "ad not found for deletion",
			slog.String("op", opDeleteAd),
			slog.Int64("ad_id", id),
		)
		return ErrAdNotFound
	}

	s.log.InfoContext(ctx, "ad soft-deleted",
		slog.String("op", opDeleteAd),
		slog.Int64("ad_id", id),
	)
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

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCloseAd),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)

	result, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to close ad",
			slog.String("op", opCloseAd),
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CloseAd: exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		s.log.WarnContext(ctx, "ad not found for closing",
			slog.String("op", opCloseAd),
			slog.Int64("ad_id", id),
		)
		return ErrAdNotFound
	}

	s.log.InfoContext(ctx, "ad archived",
		slog.String("op", opCloseAd),
		slog.Int64("ad_id", id),
	)
	return nil
}

// GetAdsByUserID возвращает список всех объявлений пользователя по его ID.
// Бустинг здесь не применяется (профиль продавца), но флаги передаём — клиент рисует бейджи.
func (s *AdStorage) GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id, p.seller_id, p.category_id, p.title, p.description,
			p.price, p.status, COALESCE(p.location, '') AS location,
			p.lat, p.lon,
			p.created_at, p.updated_at,
			COALESCE(array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL), '{}') AS photos,
			p.views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count,
			COALESCE(vpp.is_boosted, false)     AS is_boosted,
			COALESCE(vpp.is_highlighted, false) AS is_highlighted
		FROM product p
		LEFT JOIN product_image       pi  ON pi.product_id  = p.id
		LEFT JOIN favorite            f   ON f.product_id   = p.id
		LEFT JOIN v_product_promotion vpp ON vpp.product_id = p.id
		WHERE p.seller_id = $1
		  AND p.deleted_at IS NULL
		  AND p.status NOT IN ('admin_deleted', 'pending_moderation', 'rejected')
		GROUP BY p.id, vpp.is_boosted, vpp.is_highlighted
		ORDER BY p.created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetAdsByUserID),
		slog.Int64("user_id", userID),
	)

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query ads by user id",
			slog.String("op", opGetAdsByUserID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetAdsByUserID: query: %w", err)
	}

	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID, &ad.SellerID, &ad.CategoryID, &ad.Title, &ad.Description,
			&ad.Price, &ad.Status, &ad.Location, &ad.Lat, &ad.Lon, &ad.CreatedAt, &ad.UpdatedAt,
			&photos, &ad.ViewsCount, &ad.FavoritesCount,
			&ad.IsBoosted, &ad.IsHighlighted,
		); err != nil {
			s.log.ErrorContext(ctx, "failed to scan ad row",
				slog.String("op", opGetAdsByUserID),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("GetAdsByUserID: scan: %w", err)
		}

		ad.Photos = photos
		ads = append(ads, ad)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetAdsByUserID: load characteristics: %w", err)
	}

	s.log.DebugContext(ctx, "ads by user id fetched",
		slog.String("op", opGetAdsByUserID),
		slog.Int64("user_id", userID),
		slog.Int("count", len(ads)),
	)
	return ads, nil
}

// AddFavorite добавляет объявление в избранное.
func (s *AdStorage) AddFavorite(ctx context.Context, userID int64, adID int64) error {
	const query = `
		INSERT INTO favorite (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opAddFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	_, err := s.pool.Exec(ctx, query, userID, adID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to add favorite",
			slog.String("op", opAddFavorite),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("AddFavorite: %w", err)
	}

	s.log.DebugContext(ctx, "favorite added",
		slog.String("op", opAddFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// RemoveFavorite удаляет объявление из избранного.
func (s *AdStorage) RemoveFavorite(ctx context.Context, userID int64, adID int64) error {
	const query = `
		DELETE FROM favorite
		WHERE user_id = $1 AND product_id = $2
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opRemoveFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	_, err := s.pool.Exec(ctx, query, userID, adID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to remove favorite",
			slog.String("op", opRemoveFavorite),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("RemoveFavorite: %w", err)
	}

	s.log.DebugContext(ctx, "favorite removed",
		slog.String("op", opRemoveFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// GetUserFavorites возвращает полный список объявлений, которые пользователь добавил в избранное.
func (s *AdStorage) GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error) {
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
			p.lat,
			p.lon,
			p.created_at,
			p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			p.views_count,
			COUNT(DISTINCT f_all.user_id) AS favorites_count,
			COALESCE(vpp.is_boosted, false)     AS is_boosted,
			COALESCE(vpp.is_highlighted, false) AS is_highlighted
		FROM favorite f
		JOIN product p ON f.product_id = p.id
		LEFT JOIN product_image       pi    ON pi.product_id    = p.id
		LEFT JOIN favorite            f_all ON f_all.product_id = p.id
		LEFT JOIN v_product_promotion vpp   ON vpp.product_id   = p.id
		WHERE f.user_id = $1
		  AND p.deleted_at IS NULL
		  AND p.status NOT IN ('admin_deleted', 'pending_moderation', 'rejected')
		GROUP BY p.id, f.created_at, vpp.is_boosted, vpp.is_highlighted
		ORDER BY f.created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetUserFavorites),
		slog.Int64("user_id", userID),
	)

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query user favorites",
			slog.String("op", opGetUserFavorites),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetUserFavorites: query: %w", err)
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		err := rows.Scan(
			&ad.ID,
			&ad.SellerID,
			&ad.CategoryID,
			&ad.Title,
			&ad.Description,
			&ad.Price,
			&ad.Status,
			&ad.Location,
			&ad.Lat,
			&ad.Lon,
			&ad.CreatedAt,
			&ad.UpdatedAt,
			&photos,
			&ad.ViewsCount,
			&ad.FavoritesCount,
			&ad.IsBoosted,
			&ad.IsHighlighted,
		)
		if err != nil {
			s.log.ErrorContext(ctx, "failed to scan favorite row",
				slog.String("op", opGetUserFavorites),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("GetUserFavorites: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUserFavorites: rows: %w", err)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetUserFavorites: load characteristics: %w", err)
	}

	s.log.DebugContext(ctx, "user favorites fetched",
		slog.String("op", opGetUserFavorites),
		slog.Int64("user_id", userID),
		slog.Int("count", len(ads)),
	)
	return ads, nil
}

// getProductCharacteristics загружает категорийные характеристики для набора product_id.
func (s *AdStorage) getProductCharacteristics(ctx context.Context, ids []int64) (map[int64][]models.ProductCharacteristic, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT pc.product_id, cc.name, pc.value
		FROM product_characteristic pc
		JOIN category_characteristic cc ON cc.id = pc.category_characteristic_id
		WHERE pc.product_id = ANY($1)
		ORDER BY cc.sort_order
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetProductCharacteristics),
		slog.Int("ids_count", len(ids)),
	)

	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query product characteristics",
			slog.String("op", opGetProductCharacteristics),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("getProductCharacteristics: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]models.ProductCharacteristic, len(ids))
	for rows.Next() {
		var productID int64
		var pc models.ProductCharacteristic
		if err := rows.Scan(&productID, &pc.Name, &pc.Value); err != nil {
			return nil, fmt.Errorf("getProductCharacteristics: scan: %w", err)
		}
		result[productID] = append(result[productID], pc)
	}

	return result, rows.Err()
}

// getProductCustomCharacteristics загружает пользовательские характеристики для набора product_id.
func (s *AdStorage) getProductCustomCharacteristics(ctx context.Context, ids []int64) (map[int64][]models.ProductCustomCharacteristic, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT product_id, name, value
		FROM product_custom_characteristic
		WHERE product_id = ANY($1)
		ORDER BY id
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetProductCustomCharacteristics),
		slog.Int("ids_count", len(ids)),
	)

	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query product custom characteristics",
			slog.String("op", opGetProductCustomCharacteristics),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("getProductCustomCharacteristics: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]models.ProductCustomCharacteristic, len(ids))
	for rows.Next() {
		var productID int64
		var cc models.ProductCustomCharacteristic
		if err := rows.Scan(&productID, &cc.Name, &cc.Value); err != nil {
			return nil, fmt.Errorf("getProductCustomCharacteristics: scan: %w", err)
		}
		result[productID] = append(result[productID], cc)
	}

	return result, rows.Err()
}

// loadCharacteristicsForAds загружает характеристики для слайса объявлений (batch).
func (s *AdStorage) loadCharacteristicsForAds(ctx context.Context, ads []models.Ad) error {
	if len(ads) == 0 {
		return nil
	}

	ids := make([]int64, len(ads))
	for i := range ads {
		ids[i] = ads[i].ID
	}

	catChars, err := s.getProductCharacteristics(ctx, ids)
	if err != nil {
		return err
	}

	customChars, err := s.getProductCustomCharacteristics(ctx, ids)
	if err != nil {
		return err
	}

	for i := range ads {
		ads[i].CategoryCharacteristics = catChars[ads[i].ID]
		if ads[i].CategoryCharacteristics == nil {
			ads[i].CategoryCharacteristics = []models.ProductCharacteristic{}
		}
		ads[i].CustomCharacteristics = customChars[ads[i].ID]
		if ads[i].CustomCharacteristics == nil {
			ads[i].CustomCharacteristics = []models.ProductCustomCharacteristic{}
		}
	}

	return nil
}

// SetProductCharacteristics выполняет UPSERT категорийных характеристик.
// Пустой value означает удаление.
func (s *AdStorage) SetProductCharacteristics(ctx context.Context, productID int64, inputs []dto.CharacteristicInput) error {
	s.log.DebugContext(ctx, "setting product characteristics",
		slog.String("op", opSetProductCharacteristics),
		slog.Int64("product_id", productID),
		slog.Int("count", len(inputs)),
	)

	for _, inp := range inputs {
		if inp.Value == "" {
			const delQ = `DELETE FROM product_characteristic WHERE product_id = $1 AND category_characteristic_id = $2`
			if _, err := s.pool.Exec(ctx, delQ, productID, inp.CategoryCharacteristicID); err != nil {
				s.log.ErrorContext(ctx, "failed to delete characteristic",
					slog.String("op", opSetProductCharacteristics),
					slog.String("error", err.Error()),
				)
				return fmt.Errorf("SetProductCharacteristics: delete: %w", err)
			}
		} else {
			const upsertQ = `
				INSERT INTO product_characteristic (product_id, category_characteristic_id, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, category_characteristic_id)
				DO UPDATE SET value = EXCLUDED.value
			`
			if _, err := s.pool.Exec(ctx, upsertQ, productID, inp.CategoryCharacteristicID, inp.Value); err != nil {
				s.log.ErrorContext(ctx, "failed to upsert characteristic",
					slog.String("op", opSetProductCharacteristics),
					slog.String("error", err.Error()),
				)
				return fmt.Errorf("SetProductCharacteristics: upsert: %w", err)
			}
		}
	}
	return nil
}

// SetProductCustomCharacteristics выполняет UPSERT пользовательских характеристик.
// Пустой value означает удаление.
func (s *AdStorage) SetProductCustomCharacteristics(ctx context.Context, productID int64, inputs []dto.CustomCharacteristicInput) error {
	s.log.DebugContext(ctx, "setting product custom characteristics",
		slog.String("op", opSetProductCustomCharacteristics),
		slog.Int64("product_id", productID),
		slog.Int("count", len(inputs)),
	)

	for _, inp := range inputs {
		if inp.Value == "" {
			const delQ = `DELETE FROM product_custom_characteristic WHERE product_id = $1 AND name = $2`
			if _, err := s.pool.Exec(ctx, delQ, productID, inp.Name); err != nil {
				s.log.ErrorContext(ctx, "failed to delete custom characteristic",
					slog.String("op", opSetProductCustomCharacteristics),
					slog.String("error", err.Error()),
				)
				return fmt.Errorf("SetProductCustomCharacteristics: delete: %w", err)
			}
		} else {
			const upsertQ = `
				INSERT INTO product_custom_characteristic (product_id, name, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, name)
				DO UPDATE SET value = EXCLUDED.value
			`
			if _, err := s.pool.Exec(ctx, upsertQ, productID, inp.Name, inp.Value); err != nil {
				s.log.ErrorContext(ctx, "failed to upsert custom characteristic",
					slog.String("op", opSetProductCustomCharacteristics),
					slog.String("error", err.Error()),
				)
				return fmt.Errorf("SetProductCustomCharacteristics: upsert: %w", err)
			}
		}
	}
	return nil
}

// SearchAds выполняет поиск объявлений по триграммам с использованием pg_trgm.
// Поиск ведётся по title (similarity) и description (word_similarity).
// Пороги устанавливаются через SET LOCAL внутри транзакции.
// variants содержит все варианты запроса (оригинал, транслит, раскладка, синонимы).
func (s *AdStorage) SearchAds(ctx context.Context, variants []string, categoryID int64, cfg config.SearchConfig) ([]models.Ad, error) {
	s.log.DebugContext(ctx, "executing search",
		slog.String("op", opSearchAds),
		slog.Any("variants", variants),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to begin transaction",
			slog.String("op", opSearchAds),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SearchAds: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// SET LOCAL не поддерживает параметризованные запросы ($1) в PostgreSQL,
	// поэтому используем fmt.Sprintf. Значения — float64 из конфига, не пользовательский ввод.
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL pg_trgm.similarity_threshold = %f", cfg.SimilarityThreshold))
	if err != nil {
		return nil, fmt.Errorf("SearchAds: set similarity_threshold: %w", err)
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL pg_trgm.word_similarity_threshold = %f", cfg.WordSimilarityThreshold))
	if err != nil {
		return nil, fmt.Errorf("SearchAds: set word_similarity_threshold: %w", err)
	}

	const query = `
		WITH variants AS (
			SELECT unnest($1::text[]) AS term
		),
		matched AS (
			SELECT p.id,
				max(greatest(
					word_similarity(v.term, p.title),
					word_similarity(v.term, p.description)
				)) AS rank
			FROM product p
			CROSS JOIN variants v
			WHERE p.deleted_at IS NULL
			  AND p.status = 'active'
			  AND (v.term <% p.title OR v.term <% p.description)
			  AND ($3::bigint = 0 OR p.category_id = $3)
			GROUP BY p.id
			ORDER BY rank DESC
			LIMIT $2
		)
		SELECT p.id, p.seller_id, p.category_id, p.title, p.description,
			p.price, p.status, COALESCE(p.location, '') AS location,
			p.lat, p.lon,
			p.created_at, p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			COUNT(DISTINCT pv.id)        AS views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count,
			COALESCE(vpp.is_boosted, false)     AS is_boosted,
			COALESCE(vpp.is_highlighted, false) AS is_highlighted
		FROM matched m
		JOIN product p ON p.id = m.id
		LEFT JOIN product_image       pi  ON pi.product_id  = p.id
		LEFT JOIN product_view        pv  ON pv.product_id  = p.id
		LEFT JOIN favorite            f   ON f.product_id   = p.id
		LEFT JOIN v_product_promotion vpp ON vpp.product_id = p.id
		GROUP BY p.id, m.rank, vpp.is_boosted, vpp.is_highlighted
		ORDER BY
			COALESCE(vpp.is_boosted, false) DESC,
			m.rank DESC
	`

	rows, err := tx.Query(ctx, query, variants, cfg.MaxResults, categoryID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to execute search query",
			slog.String("op", opSearchAds),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SearchAds: query: %w", err)
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID, &ad.SellerID, &ad.CategoryID, &ad.Title, &ad.Description,
			&ad.Price, &ad.Status, &ad.Location, &ad.Lat, &ad.Lon, &ad.CreatedAt, &ad.UpdatedAt,
			&photos, &ad.ViewsCount, &ad.FavoritesCount,
			&ad.IsBoosted, &ad.IsHighlighted,
		); err != nil {
			s.log.ErrorContext(ctx, "failed to scan search result",
				slog.String("op", opSearchAds),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("SearchAds: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SearchAds: rows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("SearchAds: commit: %w", err)
	}

	if ads == nil {
		ads = []models.Ad{}
	}

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("SearchAds: load characteristics: %w", err)
	}

	s.log.DebugContext(ctx, "search completed",
		slog.String("op", opSearchAds),
		slog.Int("results", len(ads)),
	)
	return ads, nil
}

// GetCategoryCharacteristics возвращает определения характеристик для категории.
func (s *AdStorage) GetCategoryCharacteristics(ctx context.Context, categoryID int64) ([]models.CategoryCharacteristic, error) {
	const query = `
		SELECT id, category_id, name, allowed_values, sort_order
		FROM category_characteristic
		WHERE category_id = $1
		ORDER BY sort_order
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetCategoryCharacteristics),
		slog.Int64("category_id", categoryID),
	)

	rows, err := s.pool.Query(ctx, query, categoryID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query category characteristics",
			slog.String("op", opGetCategoryCharacteristics),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetCategoryCharacteristics: %w", err)
	}
	defer rows.Close()

	var chars []models.CategoryCharacteristic
	for rows.Next() {
		var c models.CategoryCharacteristic
		if err := rows.Scan(&c.ID, &c.CategoryID, &c.Name, &c.AllowedValues, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("GetCategoryCharacteristics: scan: %w", err)
		}
		chars = append(chars, c)
	}

	if chars == nil {
		chars = []models.CategoryCharacteristic{}
	}

	s.log.DebugContext(ctx, "category characteristics fetched",
		slog.String("op", opGetCategoryCharacteristics),
		slog.Int64("category_id", categoryID),
		slog.Int("count", len(chars)),
	)
	return chars, rows.Err()
}

// UpdateAdStatus меняет status объявления без проверки владельца.
// Вызывается gRPC из Commerce при покупке: active → reserved → sold.
// Возвращает предыдущий статус.
func (s *AdStorage) UpdateAdStatus(ctx context.Context, id int64, newStatus string) (string, error) {
	const query = `
		UPDATE product
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING (
			SELECT status FROM product WHERE id = $2 AND deleted_at IS NULL
		)
	`
	// Простой подход: сначала читаем prev, потом UPDATE. RETURNING с подзапросом
	// в одном UPDATE возвращает уже новое значение. Делаем двумя запросами.
	var prevStatus string
	err := s.pool.QueryRow(ctx, `SELECT status FROM product WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&prevStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAdNotFound
		}
		return "", fmt.Errorf("UpdateAdStatus: select: %w", err)
	}

	res, err := s.pool.Exec(ctx, `UPDATE product SET status = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, newStatus, id)
	if err != nil {
		return "", fmt.Errorf("UpdateAdStatus: update: %w", err)
	}
	if res.RowsAffected() == 0 {
		return "", ErrAdNotFound
	}
	_ = query // silence unused-const in case future refactor
	return prevStatus, nil
}

// GetPriceHistory возвращает историю изменения цены объявления, отсортированную по времени.
func (s *AdStorage) GetPriceHistory(ctx context.Context, adID int64) ([]models.PricePoint, error) {
	const query = `
		SELECT price, changed_at
		FROM product_price_history
		WHERE product_id = $1
		ORDER BY changed_at ASC;
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetPriceHistory),
		slog.Int64("ad_id", adID),
	)

	rows, err := s.pool.Query(ctx, query, adID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query price history",
			slog.String("op", opGetPriceHistory),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetPriceHistory: query: %w", err)
	}
	defer rows.Close()

	var history []models.PricePoint
	for rows.Next() {
		var pp models.PricePoint
		if err := rows.Scan(&pp.Price, &pp.ChangedAt); err != nil {
			s.log.ErrorContext(ctx, "failed to scan price point",
				slog.String("op", opGetPriceHistory),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("GetPriceHistory: scan: %w", err)
		}
		history = append(history, pp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetPriceHistory: rows: %w", err)
	}

	return history, nil
}

// AdminDeleteAd выполняет жёсткое (для продавца — невидимое) удаление от имени админа:
// проставляет status='admin_deleted' и заполняет deleted_by_admin_id.
// Возвращает seller_id и title удалённого объявления (нужны для системного сообщения).
func (s *AdStorage) AdminDeleteAd(ctx context.Context, adID, adminID int64) (sellerID int64, title string, err error) {
	const query = `
		UPDATE product
		SET status              = 'admin_deleted',
		    deleted_by_admin_id = $2,
		    deleted_at          = NOW(),
		    updated_at          = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status <> 'admin_deleted'
		RETURNING seller_id, title
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opAdminDeleteAd),
		slog.Int64("ad_id", adID),
		slog.Int64("admin_id", adminID),
	)

	err = s.pool.QueryRow(ctx, query, adID, adminID).Scan(&sellerID, &title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrAdNotFound
		}
		s.log.ErrorContext(ctx, "failed to admin-delete ad",
			slog.String("op", opAdminDeleteAd),
			slog.Int64("ad_id", adID),
			slog.String("error", err.Error()),
		)
		return 0, "", fmt.Errorf("AdminDeleteAd: %w", err)
	}

	s.log.InfoContext(ctx, "ad admin-deleted",
		slog.String("op", opAdminDeleteAd),
		slog.Int64("ad_id", adID),
	)
	return sellerID, title, nil
}

// ModerateAd переводит объявление из pending_moderation в active или rejected.
// Для rejected сохраняет причину. Возвращает seller_id и title для системного сообщения.
func (s *AdStorage) ModerateAd(
	ctx context.Context,
	adID int64,
	newStatus string,
	reason string,
) (sellerID int64, title string, err error) {
	const query = `
		UPDATE product
		SET status            = $2,
		    rejection_reason  = CASE WHEN $2 = 'rejected' THEN $3 ELSE NULL END,
		    updated_at        = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status = 'pending_moderation'
		RETURNING seller_id, title
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opModerateAd),
		slog.Int64("ad_id", adID),
		slog.String("new_status", newStatus),
	)

	err = s.pool.QueryRow(ctx, query, adID, newStatus, reason).Scan(&sellerID, &title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrAdNotFound
		}
		s.log.ErrorContext(ctx, "failed to moderate ad",
			slog.String("op", opModerateAd),
			slog.Int64("ad_id", adID),
			slog.String("error", err.Error()),
		)
		return 0, "", fmt.Errorf("ModerateAd: %w", err)
	}

	s.log.InfoContext(ctx, "ad moderation applied",
		slog.String("op", opModerateAd),
		slog.Int64("ad_id", adID),
		slog.String("new_status", newStatus),
	)
	return sellerID, title, nil
}

// GetModerationQueue возвращает объявления, ожидающие модерации.
// Сортировка по дате создания (старые сверху — обрабатываются первыми).
func (s *AdStorage) GetModerationQueue(ctx context.Context) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id, p.seller_id, p.category_id, p.title, p.description,
			p.price, p.status, COALESCE(p.location, '') AS location,
			p.lat, p.lon,
			p.created_at, p.updated_at,
			COALESCE(array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL), '{}') AS photos,
			p.views_count,
			0::bigint AS favorites_count,
			false AS is_boosted,
			false AS is_highlighted
		FROM product p
		LEFT JOIN product_image pi ON pi.product_id = p.id
		WHERE p.deleted_at IS NULL
		  AND p.status = 'pending_moderation'
		GROUP BY p.id
		ORDER BY p.created_at ASC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetModerationQueue),
	)

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetModerationQueue: query: %w", err)
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID, &ad.SellerID, &ad.CategoryID, &ad.Title, &ad.Description,
			&ad.Price, &ad.Status, &ad.Location, &ad.Lat, &ad.Lon,
			&ad.CreatedAt, &ad.UpdatedAt,
			&photos, &ad.ViewsCount, &ad.FavoritesCount,
			&ad.IsBoosted, &ad.IsHighlighted,
		); err != nil {
			return nil, fmt.Errorf("GetModerationQueue: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if ads == nil {
		ads = []models.Ad{}
	}
	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetModerationQueue: load characteristics: %w", err)
	}
	return ads, nil
}

// GetUserAdsByStatus возвращает объявления пользователя с указанным статусом.
// Используется для вкладки «На модерации» в профиле продавца.
func (s *AdStorage) GetUserAdsByStatus(ctx context.Context, userID int64, status string) ([]models.Ad, error) {
	const query = `
		SELECT
			p.id, p.seller_id, p.category_id, p.title, p.description,
			p.price, p.status, COALESCE(p.location, '') AS location,
			p.lat, p.lon,
			p.created_at, p.updated_at,
			COALESCE(array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL), '{}') AS photos,
			p.views_count,
			COUNT(DISTINCT f.product_id) AS favorites_count,
			false AS is_boosted,
			false AS is_highlighted
		FROM product p
		LEFT JOIN product_image pi ON pi.product_id = p.id
		LEFT JOIN favorite       f  ON f.product_id  = p.id
		WHERE p.seller_id = $1
		  AND p.deleted_at IS NULL
		  AND p.status = $2
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetUserAdsByStatus),
		slog.Int64("user_id", userID),
		slog.String("status", status),
	)

	rows, err := s.pool.Query(ctx, query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("GetUserAdsByStatus: query: %w", err)
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var ad models.Ad
		var photos []string
		if err := rows.Scan(
			&ad.ID, &ad.SellerID, &ad.CategoryID, &ad.Title, &ad.Description,
			&ad.Price, &ad.Status, &ad.Location, &ad.Lat, &ad.Lon,
			&ad.CreatedAt, &ad.UpdatedAt,
			&photos, &ad.ViewsCount, &ad.FavoritesCount,
			&ad.IsBoosted, &ad.IsHighlighted,
		); err != nil {
			return nil, fmt.Errorf("GetUserAdsByStatus: scan: %w", err)
		}
		ad.Photos = photos
		ads = append(ads, ad)
	}

	if ads == nil {
		ads = []models.Ad{}
	}
	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetUserAdsByStatus: load characteristics: %w", err)
	}
	return ads, nil
}
