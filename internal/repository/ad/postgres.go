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

	// Подгрузка характеристик
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

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetAllAds: load characteristics: %w", err)
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

// DeleteProductImages удаляет все изображения объявления.
func (s *AdStorage) DeleteProductImages(ctx context.Context, adID int64) error {
	const query = `DELETE FROM product_image WHERE product_id = $1`

	_, err := s.pool.Exec(ctx, query, adID)
	if err != nil {
		return fmt.Errorf("DeleteProductImages: exec: %w", err)
	}

	return nil
}

// UpdateAd обновляет объявление. Проверяет принадлежность объявления пользователю.
// Обновляет только те поля, которые были отправлены (не nil).
func (s *AdStorage) UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error {
	var updates []string
	var args []any

	argNum := 1

	// Динамически строим SET clause с только переданными полями
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

	// Если нет полей для обновления, возвращаем ошибку
	if len(updates) == 0 {
		return fmt.Errorf("UpdateAd: no fields to update")
	}

	// Добавляем updated_at и WHERE условия
	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE product
		SET %s
		WHERE id = $%d
		  AND seller_id = $%d
		  AND deleted_at IS NULL
	`, strings.Join(updates, ", "), argNum, argNum+1)

	args = append(args, req.ID, req.UserID)

	result, err := s.pool.Exec(ctx, query, args...)
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
			p.price, p.status, COALESCE(p.location, '') AS location, p.created_at, p.updated_at,
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
			&ad.Price, &ad.Status, &ad.Location, &ad.CreatedAt, &ad.UpdatedAt,
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

	if err := s.loadCharacteristicsForAds(ctx, ads); err != nil {
		return nil, fmt.Errorf("GetAdsByUserID: load characteristics: %w", err)
	}

	return ads, nil
}

// AddFavorite добавляет объявление в избранное.
func (s *AdStorage) AddFavorite(ctx context.Context, userID int64, adID int64) error {
	// ON CONFLICT DO NOTHING чтобы не возвращать ошибку, если пользователь нажал "лайк" дважды
	const query = `
		INSERT INTO favorite (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	_, err := s.pool.Exec(ctx, query, userID, adID)
	if err != nil {
		return fmt.Errorf("AddFavorite: %w", err)
	}

	return nil
}

// RemoveFavorite удаляет объявление из избранного.
func (s *AdStorage) RemoveFavorite(ctx context.Context, userID int64, adID int64) error {
	const query = `
		DELETE FROM favorite
		WHERE user_id = $1 AND product_id = $2
	`

	_, err := s.pool.Exec(ctx, query, userID, adID)
	if err != nil {
		return fmt.Errorf("RemoveFavorite: %w", err)
	}

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
			p.created_at,
			p.updated_at,
			COALESCE(
				array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL),
				'{}'
			) AS photos,
			COUNT(DISTINCT pv.id)        AS views_count,
			COUNT(DISTINCT f_all.user_id) AS favorites_count
		FROM favorite f
		JOIN product p ON f.product_id = p.id
		LEFT JOIN product_image pi ON pi.product_id = p.id
		LEFT JOIN product_view  pv ON pv.product_id = p.id
		LEFT JOIN favorite   f_all ON f_all.product_id = p.id
		WHERE f.user_id = $1
		  AND p.deleted_at IS NULL
		GROUP BY p.id, f.created_at
		ORDER BY f.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
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
			&ad.CreatedAt,
			&ad.UpdatedAt,
			&photos,
			&ad.ViewsCount,
			&ad.FavoritesCount,
		)
		if err != nil {
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

	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
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

	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
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
	for _, inp := range inputs {
		if inp.Value == "" {
			// Удаление
			const delQ = `DELETE FROM product_characteristic WHERE product_id = $1 AND category_characteristic_id = $2`
			if _, err := s.pool.Exec(ctx, delQ, productID, inp.CategoryCharacteristicID); err != nil {
				return fmt.Errorf("SetProductCharacteristics: delete: %w", err)
			}
		} else {
			// UPSERT
			const upsertQ = `
				INSERT INTO product_characteristic (product_id, category_characteristic_id, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, category_characteristic_id)
				DO UPDATE SET value = EXCLUDED.value
			`
			if _, err := s.pool.Exec(ctx, upsertQ, productID, inp.CategoryCharacteristicID, inp.Value); err != nil {
				return fmt.Errorf("SetProductCharacteristics: upsert: %w", err)
			}
		}
	}
	return nil
}

// SetProductCustomCharacteristics выполняет UPSERT пользовательских характеристик.
// Пустой value означает удаление.
func (s *AdStorage) SetProductCustomCharacteristics(ctx context.Context, productID int64, inputs []dto.CustomCharacteristicInput) error {
	for _, inp := range inputs {
		if inp.Value == "" {
			// Удаление
			const delQ = `DELETE FROM product_custom_characteristic WHERE product_id = $1 AND name = $2`
			if _, err := s.pool.Exec(ctx, delQ, productID, inp.Name); err != nil {
				return fmt.Errorf("SetProductCustomCharacteristics: delete: %w", err)
			}
		} else {
			// UPSERT
			const upsertQ = `
				INSERT INTO product_custom_characteristic (product_id, name, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, name)
				DO UPDATE SET value = EXCLUDED.value
			`
			if _, err := s.pool.Exec(ctx, upsertQ, productID, inp.Name, inp.Value); err != nil {
				return fmt.Errorf("SetProductCustomCharacteristics: upsert: %w", err)
			}
		}
	}
	return nil
}

// GetCategoryCharacteristics возвращает определения характеристик для категории.
func (s *AdStorage) GetCategoryCharacteristics(ctx context.Context, categoryID int64) ([]models.CategoryCharacteristic, error) {
	const query = `
		SELECT id, category_id, name, allowed_values, sort_order
		FROM category_characteristic
		WHERE category_id = $1
		ORDER BY sort_order
	`

	rows, err := s.pool.Query(ctx, query, categoryID)
	if err != nil {
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

	return chars, rows.Err()
}
