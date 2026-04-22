package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opGetOrCreateChat  = "db.chat.GetOrCreateChat"
	opCreateMessage    = "db.chat.CreateMessage"
	opGetChatByID      = "db.chat.GetChatByID"
	opCompletePurchase = "db.chat.CompletePurchase"
)

var ErrChatNotFound = errors.New("chat not found")

// PgxPool интерфейс для пула соединений с поддержкой транзакций,
// позволяющий подменять его моком в тестах.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ChatStorage отвечает за операции с чатами и сообщениями.
type ChatStorage struct {
	pool PgxPool
	log  *slog.Logger
}

func NewChatStorage(pool PgxPool, log *slog.Logger) *ChatStorage {
	return &ChatStorage{
		pool: pool,
		log:  log,
	}
}

// GetOrCreateChat получает или создает чат между покупателем и продавцом.
// Возвращает ID чата.
func (cs *ChatStorage) GetOrCreateChat(
	ctx context.Context,
	adID int64,
	buyerID int64,
	sellerID int64,
) (int64, error) {
	var chatID int64

	const querySelect = `
		SELECT id FROM chat
		WHERE product_id = $1 AND buyer_id = $2 AND seller_id = $3
		LIMIT 1
	`

	// Сначала пытаемся найти существующий чат
	err := cs.pool.QueryRow(ctx, querySelect, adID, buyerID, sellerID).Scan(&chatID)
	if err == nil {
		// Чат уже существует
		return chatID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		cs.log.ErrorContext(ctx, "failed to select chat",
			slog.String("op", opGetOrCreateChat),
			slog.String("error", err.Error()),
		)
		return 0, err
	}

	const queryInsert = `
		INSERT INTO chat (product_id, buyer_id, seller_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	// Чата нет, создаем новый
	err = cs.pool.QueryRow(ctx, queryInsert, adID, buyerID, sellerID).Scan(&chatID)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to create chat",
			slog.String("op", opGetOrCreateChat),
			slog.String("error", err.Error()),
		)
		return 0, err
	}

	return chatID, nil
}

// GetChatByID возвращает чат по его ID. Возвращает ErrChatNotFound, если чата нет.
func (cs *ChatStorage) GetChatByID(ctx context.Context, chatID int64) (models.Chat, error) {
	const query = `
		SELECT id, product_id, buyer_id, seller_id, created_at, updated_at
		FROM chat
		WHERE id = $1
	`

	var chat models.Chat
	err := cs.pool.QueryRow(ctx, query, chatID).Scan(
		&chat.ID,
		&chat.AdID,
		&chat.BuyerID,
		&chat.SellerID,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Chat{}, ErrChatNotFound
		}
		cs.log.ErrorContext(ctx, "failed to get chat by id",
			slog.String("op", opGetChatByID),
			slog.String("error", err.Error()),
		)
		return models.Chat{}, err
	}

	return chat, nil
}

// CreateMessage создает новое сообщение в чате.
func (cs *ChatStorage) CreateMessage(ctx context.Context, message *models.Message) (int64, error) {
	var msgID int64

	const query = `
		INSERT INTO message (chat_id, sender_id, text_content, msg_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := cs.pool.QueryRow(
		ctx,
		query,
		message.ChatID,
		message.SenderID,
		message.Text,
		message.Type,
	).Scan(&msgID)

	if err != nil {
		cs.log.ErrorContext(ctx, "failed to create message",
			slog.String("op", opCreateMessage),
			slog.String("error", err.Error()),
		)
		return 0, err
	}

	return msgID, nil
}

// CompletePurchase атомарно завершает сделку: создает заказ с позицией,
// переводит товар в статус 'sold' и удаляет его из корзин всех пользователей.
func (cs *ChatStorage) CompletePurchase(
	ctx context.Context,
	buyerID int64,
	productID int64,
	price int64,
) error {
	tx, err := cs.pool.Begin(ctx)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to begin transaction",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID int64
	const createOrderQuery = `
		INSERT INTO "order" (buyer_id, total_amount, status)
		VALUES ($1, $2, 'completed')
		RETURNING id
	`

	// Создаем заказ
	if err = tx.QueryRow(ctx, createOrderQuery, buyerID, price).Scan(&orderID); err != nil {
		cs.log.ErrorContext(ctx, "failed to create order",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: create order: %w", err)
	}

	const createItemQuery = `
		INSERT INTO order_item (order_id, product_id, price_at_purchase, quantity)
		VALUES ($1, $2, $3, 1)
	`

	// Создаем позицию заказа
	if _, err = tx.Exec(ctx, createItemQuery, orderID, productID, price); err != nil {
		cs.log.ErrorContext(ctx, "failed to create order item",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: create order item: %w", err)
	}

	const updateProductQuery = `
		UPDATE product SET status = 'sold', updated_at = NOW()
		WHERE id = $1
	`

	// Обновляем статус товара
	res, err := tx.Exec(ctx, updateProductQuery, productID)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to update product status",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: update product: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("ad not found")
	}

	const clearCartsQuery = `DELETE FROM cart_item WHERE product_id = $1`

	// Удаляем товар из всех корзин
	if _, err = tx.Exec(ctx, clearCartsQuery, productID); err != nil {
		cs.log.ErrorContext(ctx, "failed to clear product from carts",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: clear carts: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		cs.log.ErrorContext(ctx, "failed to commit transaction",
			slog.String("op", opCompletePurchase),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("CompletePurchase: commit: %w", err)
	}

	return nil
}
