package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	ErrProductReserved      = errors.New("one or more products are no longer available")
	ErrProductAlreadyInCart = errors.New("product already in cart")
	ErrCartEmpty            = errors.New("cart is empty")
)

// CartStorage отвечает за операции с корзиной.
type CartStorage struct {
	pool PgxPoolTx
}

func NewCartStorage(pool PgxPoolTx) *CartStorage {
	return &CartStorage{pool: pool}
}

// Add добавляет товар в корзину пользователя.
func (s *CartStorage) Add(ctx context.Context, userID, productID int64) error {
	// Добавляем, игнорируя конфликт (если товар уже в корзине - ничего страшного,
	// или можно возвращать ошибку. По дизайну "вернуть ошибку".
	// В PostgreSQL добавим ON CONFLICT DO NOTHING и проверим RowsAffected.
	const query = `
		INSERT INTO cart_item (user_id, product_id, quantity)
		VALUES ($1, $2, 1)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`

	res, err := s.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		return fmt.Errorf("CartStorage.Add: exec: %w", err)
	}

	if res.RowsAffected() == 0 {
		return ErrProductAlreadyInCart
	}

	return nil
}

// Remove удаляет товар из корзины.
func (s *CartStorage) Remove(ctx context.Context, userID, productID int64) error {
	const query = `
		DELETE FROM cart_item
		WHERE user_id = $1 AND product_id = $2
	`

	res, err := s.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		return fmt.Errorf("CartStorage.Remove: exec: %w", err)
	}

	if res.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}

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

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
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

	return items, nil
}

// Clear полностью очищает корзину пользователя.
func (s *CartStorage) Clear(ctx context.Context, userID int64) error {
	const query = `DELETE FROM cart_item WHERE user_id = $1`
	_, err := s.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("CartStorage.Clear: exec: %w", err)
	}
	return nil
}

// Checkout оформляет заказ на все товары в корзине.
// Возвращает список ID заказов и контакты продавцов.
func (s *CartStorage) Checkout(ctx context.Context, buyerID int64) ([]int64, map[int64]*dto.SellerContact, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("Checkout: begin tx: %w", err)
	}
	// Делаем Rollback с обработкой паники
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Получаем все товары в корзине с эксклюзивной блокировкой (FOR UPDATE),
	// чтобы никто другой не мог их изменить до завершения транзакции.
	// Очередь сортировки помогает избежать DeadLock.
	const queryItems = `
		SELECT p.id, p.seller_id, p.price, p.status, u.id, u.first_name, u.email
		FROM cart_item c
		JOIN product p ON c.product_id = p.id
		JOIN "user" u ON p.seller_id = u.id
		WHERE c.user_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.id FOR UPDATE OF p
	`

	rows, err := tx.Query(ctx, queryItems, buyerID)
	if err != nil {
		return nil, nil, fmt.Errorf("Checkout: query cart items: %w", err)
	}

	type cartRow struct {
		productID int64
		sellerID  int64
		price     int64
		status    string
		seller    dto.SellerContact
	}

	var items []cartRow
	for rows.Next() {
		var row cartRow
		if err := rows.Scan(
			&row.productID, &row.sellerID, &row.price, &row.status,
			&row.seller.ID, &row.seller.Name, &row.seller.Email,
		); err != nil {
			rows.Close()
			return nil, nil, fmt.Errorf("Checkout: scan cart item: %w", err)
		}
		items = append(items, row)
	}
	rows.Close()

	if len(items) == 0 {
		return nil, nil, ErrCartEmpty
	}

	// 2. Проверяем статус всех товаров
	for _, item := range items {
		if item.status != "active" {
			// Откат транзакции произойдет автоматически из-за defer tx.Rollback
			return nil, nil, ErrProductReserved
		}
	}

	// 3. Группируем товары по продавцам для создания заказов
	ordersBySeller := make(map[int64][]cartRow)
	sellers := make(map[int64]*dto.SellerContact)

	for _, item := range items {
		ordersBySeller[item.sellerID] = append(ordersBySeller[item.sellerID], item)
		if _, exists := sellers[item.sellerID]; !exists {
			seller := item.seller
			sellers[item.sellerID] = &seller
		}
	}

	var orderIDs []int64

	// 4. Создаем заказы
	for sellerID, sellerItems := range ordersBySeller {
		var totalAmount int64
		for _, item := range sellerItems {
			totalAmount += item.price
		}

		var orderID int64
		const createOrderQuery = `
			INSERT INTO "order" (buyer_id, total_amount, status)
			VALUES ($1, $2, 'created')
			RETURNING id
		`
		err = tx.QueryRow(ctx, createOrderQuery, buyerID, totalAmount).Scan(&orderID)
		if err != nil {
			return nil, nil, fmt.Errorf("Checkout: create order for seller %d: %w", sellerID, err)
		}

		orderIDs = append(orderIDs, orderID)

		// 5. Создаем элементы заказа
		for _, item := range sellerItems {
			const createOrderItemQuery = `
				INSERT INTO order_item (order_id, product_id, price_at_purchase, quantity)
				VALUES ($1, $2, $3, 1)
			`
			_, err = tx.Exec(ctx, createOrderItemQuery, orderID, item.productID, item.price)
			if err != nil {
				return nil, nil, fmt.Errorf("Checkout: create order item %d: %w", item.productID, err)
			}

			// 6. Обновляем статус товара
			const updateProductQuery = `
				UPDATE product SET status = 'reserved', updated_at = NOW()
				WHERE id = $1
			`
			_, err = tx.Exec(ctx, updateProductQuery, item.productID)
			if err != nil {
				return nil, nil, fmt.Errorf("Checkout: update product status %d: %w", item.productID, err)
			}
		}
	}

	// 7. Удаляем все товары из корзины данного пользователя
	const clearCartQuery = `DELETE FROM cart_item WHERE user_id = $1`
	_, err = tx.Exec(ctx, clearCartQuery, buyerID)
	if err != nil {
		return nil, nil, fmt.Errorf("Checkout: clear cart: %w", err)
	}

	// 8. Комитим транзакцию
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("Checkout: commit tx: %w", err)
	}

	return orderIDs, sellers, nil
}
