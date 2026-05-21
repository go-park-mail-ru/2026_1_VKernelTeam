// Package testreset реализует низкоуровневую очистку всех данных,
// принадлежащих пользователю, в одной SQL-транзакции.
//
// Используется только тестовой ручкой POST /api/v1/test/reset.
package testreset

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const opWipeUserData = "db.testreset.WipeUserData"

// Repository хранит пул соединений PostgreSQL и пишет в БД через явную транзакцию.
type Repository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// New создаёт репозиторий очистки данных пользователя.
func New(pool *pgxpool.Pool, log *slog.Logger) *Repository {
	return &Repository{pool: pool, log: log}
}

// step описывает один шаг очистки в составе общей транзакции.
type step struct {
	name string
	sql  string
}

// wipeSteps перечисляет операции в порядке, безопасном для FK с RESTRICT/CASCADE.
// Все шаги принимают единственный параметр $1 — userID.
var wipeSteps = []step{
	{"cart_item", `DELETE FROM cart_item WHERE user_id = $1`},
	{"favorite", `DELETE FROM favorite WHERE user_id = $1`},
	{"product_view", `DELETE FROM product_view WHERE user_id = $1`},
	{"review", `DELETE FROM review WHERE sender_id = $1 OR receiver_id = $1`},
	{"message", `DELETE FROM message WHERE sender_id = $1`},
	{"chat", `DELETE FROM chat WHERE buyer_id = $1 OR seller_id = $1`},
	{"support_message", `DELETE FROM support_message WHERE user_id = $1`},
	{"support_ticket", `DELETE FROM support_ticket WHERE user_id = $1`},
	{"wallet_transaction", `DELETE FROM wallet_transaction WHERE user_id = $1`},
	{"payment", `DELETE FROM payment WHERE user_id = $1`},
	{"promotion", `DELETE FROM promotion WHERE user_id = $1`},
	{"wallet", `DELETE FROM wallet WHERE user_id = $1`},
	{"order_item_buyer", `DELETE FROM order_item WHERE order_id IN (SELECT id FROM "order" WHERE buyer_id = $1)`},
	{"order_buyer", `DELETE FROM "order" WHERE buyer_id = $1`},
	{"order_item_seller", `DELETE FROM order_item WHERE product_id IN (SELECT id FROM product WHERE seller_id = $1)`},
	{"product", `DELETE FROM product WHERE seller_id = $1`},
	{"user_reset", `UPDATE "user" SET avatar_path = NULL, rating = 0 WHERE id = $1`},
}

// WipeUserData выполняет все шаги очистки в одной транзакции.
// При ошибке любого шага транзакция откатывается целиком.
func (r *Repository) WipeUserData(ctx context.Context, userID int64) error {
	r.log.InfoContext(ctx, "wipe user data start",
		slog.String("op", opWipeUserData),
		slog.Int64("user_id", userID),
	)

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", opWipeUserData, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, s := range wipeSteps {
		tag, err := tx.Exec(ctx, s.sql, userID)
		if err != nil {
			r.log.ErrorContext(ctx, "wipe step failed",
				slog.String("op", opWipeUserData),
				slog.String("step", s.name),
				slog.Int64("user_id", userID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: step %q: %w", opWipeUserData, s.name, err)
		}
		r.log.DebugContext(ctx, "wipe step done",
			slog.String("op", opWipeUserData),
			slog.String("step", s.name),
			slog.Int64("rows", tag.RowsAffected()),
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit: %w", opWipeUserData, err)
	}

	r.log.InfoContext(ctx, "wipe user data done",
		slog.String("op", opWipeUserData),
		slog.Int64("user_id", userID),
	)
	return nil
}
