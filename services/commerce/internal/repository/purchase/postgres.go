// Package purchase — хранилище покупок текущего пользователя (GET /profile/purchases).
// Источник данных — таблица "order" + order_item + product + chat.
package purchase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

const (
	opListByBuyer = "db.purchase.ListByBuyer"
)

// PgxPool — минимальный контракт пула pgx (под pgxmock в тестах).
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// PurchaseStorage — read-only выборка покупок.
type PurchaseStorage struct {
	pool PgxPool
	log  *slog.Logger
}

// NewStorage создаёт хранилище покупок поверх пула pgx.
func NewStorage(pool PgxPool, log *slog.Logger) *PurchaseStorage {
	return &PurchaseStorage{pool: pool, log: log}
}

// purchaseSelect — общий SELECT-фрагмент. WHERE/ORDER/LIMIT добавляются на месте.
// Сортировка по o.id DESC даёт строгую "по времени покупки" в обратном порядке,
// так как id у "order" монотонно растёт во времени.
const purchaseSelect = `
		SELECT
		    o.id, oi.product_id,
		    p.title, oi.price_at_purchase,
		    COALESCE(
		        (SELECT file_path FROM product_image
		         WHERE product_id = p.id ORDER BY sort_order ASC LIMIT 1),
		        ''
		    ) AS photo,
		    COALESCE(p.location, '') AS location,
		    u.id, u.first_name, COALESCE(u.avatar_path, ''),
		    o.source, o.created_at, o.chat_id
		FROM "order" o
		JOIN order_item oi ON oi.order_id  = o.id
		JOIN product    p  ON p.id         = oi.product_id
		JOIN "user"     u  ON u.id         = p.seller_id`

// ListByBuyer возвращает страницу покупок для buyerID c курсорной пагинацией по o.id.
// cursor=nil — первая страница; cursor=N — следующая (o.id < N).
// Запрашивается limit штук; usecase делает look-ahead за счёт limit+1.
func (s *PurchaseStorage) ListByBuyer(
	ctx context.Context, buyerID int64, cursor *int64, limit int,
) ([]dto.PurchaseItem, error) {
	const query = purchaseSelect + `
		WHERE o.buyer_id = $1
		  AND ($2::bigint IS NULL OR o.id < $2)
		ORDER BY o.id DESC
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, buyerID, cursor, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query purchases",
			slog.String("op", opListByBuyer),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("PurchaseStorage.ListByBuyer: %w", err)
	}
	defer rows.Close()

	out := []dto.PurchaseItem{}
	for rows.Next() {
		var item dto.PurchaseItem
		if err := rows.Scan(
			&item.OrderID, &item.ProductID,
			&item.Title, &item.Price,
			&item.Photo,
			&item.Location,
			&item.Seller.ID, &item.Seller.Name, &item.Seller.AvatarPath,
			&item.Source, &item.PurchasedAt, &item.ChatID,
		); err != nil {
			return nil, fmt.Errorf("PurchaseStorage.ListByBuyer: scan: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PurchaseStorage.ListByBuyer: rows: %w", err)
	}
	return out, nil
}
