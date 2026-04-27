package cart

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opCartAdd       = "db.cart.Add"
	opCartRemove    = "db.cart.Remove"
	opCartGetByUser = "db.cart.GetByUserID"
	opCartClear     = "db.cart.Clear"
)

// PgxPoolTx интерфейс для пула соединений с поддержкой транзакций
type PgxPoolTx interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

var (
	ErrCartItemNotFound     = errors.New("cart item not found")
	ErrProductNotFound      = errors.New("product not found or not active")
	ErrOwnProduct           = errors.New("cannot add own product to cart")
	ErrProductAlreadyInCart = errors.New("product already in cart")
)

// CartStorage отвечает за операции с корзиной.
type CartStorage struct {
	pool PgxPoolTx
	log  *slog.Logger
}

func NewCartStorage(pool PgxPoolTx, log *slog.Logger) *CartStorage {
	return &CartStorage{pool: pool, log: log}
}

// Add добавляет товар в корзину пользователя.
func (s *CartStorage) Add(ctx context.Context, userID, productID int64) error {
	const query = `
		INSERT INTO cart_item (user_id, product_id, quantity)
		VALUES ($1, $2, 1)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCartAdd),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)

	res, err := s.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to add to cart",
			slog.String("op", opCartAdd),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CartStorage.Add: exec: %w", err)
	}

	if res.RowsAffected() == 0 {
		s.log.DebugContext(ctx, "product already in cart",
			slog.String("op", opCartAdd),
			slog.Int64("user_id", userID),
			slog.Int64("product_id", productID),
		)
		return ErrProductAlreadyInCart
	}

	s.log.DebugContext(ctx, "product added to cart",
		slog.String("op", opCartAdd),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)
	return nil
}

// Remove удаляет товар из корзины.
func (s *CartStorage) Remove(ctx context.Context, userID, productID int64) error {
	const query = `
		DELETE FROM cart_item
		WHERE user_id = $1 AND product_id = $2
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCartRemove),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)

	res, err := s.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to remove from cart",
			slog.String("op", opCartRemove),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CartStorage.Remove: exec: %w", err)
	}

	if res.RowsAffected() == 0 {
		s.log.DebugContext(ctx, "cart item not found",
			slog.String("op", opCartRemove),
			slog.Int64("user_id", userID),
			slog.Int64("product_id", productID),
		)
		return ErrCartItemNotFound
	}

	s.log.DebugContext(ctx, "product removed from cart",
		slog.String("op", opCartRemove),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)
	return nil
}

// GetByUserID возвращает все товары в корзине пользователя с актуальными данными о продукте.
func (s *CartStorage) GetByUserID(ctx context.Context, userID int64) ([]dto.CartItemResponse, error) {
	const query = `
		SELECT
			p.id, p.title, p.price, p.seller_id, u.first_name,
			(
				SELECT file_path FROM product_image
				WHERE product_id = p.id
				ORDER BY sort_order ASC LIMIT 1
			) AS image_path
		FROM cart_item c
		JOIN product p ON c.product_id = p.id
		JOIN "user" u ON p.seller_id = u.id
		WHERE c.user_id = $1 AND p.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCartGetByUser),
		slog.Int64("user_id", userID),
	)

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get cart items",
			slog.String("op", opCartGetByUser),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("CartStorage.GetByUserID: query: %w", err)
	}
	defer rows.Close()

	var items []dto.CartItemResponse
	for rows.Next() {
		var item dto.CartItemResponse
		var imagePath *string
		if err := rows.Scan(
			&item.ProductID, &item.Title, &item.Price, &item.SellerID, &item.SellerName, &imagePath,
		); err != nil {
			s.log.ErrorContext(ctx, "failed to scan cart item",
				slog.String("op", opCartGetByUser),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("CartStorage.GetByUserID: scan: %w", err)
		}
		if imagePath != nil {
			item.ImagePath = *imagePath
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CartStorage.GetByUserID: rows: %w", err)
	}

	if items == nil {
		items = []dto.CartItemResponse{}
	}

	s.log.DebugContext(ctx, "cart items fetched",
		slog.String("op", opCartGetByUser),
		slog.Int64("user_id", userID),
		slog.Int("count", len(items)),
	)
	return items, nil
}

// Clear полностью очищает корзину пользователя.
func (s *CartStorage) Clear(ctx context.Context, userID int64) error {
	const query = `DELETE FROM cart_item WHERE user_id = $1`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCartClear),
		slog.Int64("user_id", userID),
	)

	_, err := s.pool.Exec(ctx, query, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to clear cart",
			slog.String("op", opCartClear),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CartStorage.Clear: exec: %w", err)
	}

	s.log.DebugContext(ctx, "cart cleared",
		slog.String("op", opCartClear),
		slog.Int64("user_id", userID),
	)
	return nil
}

