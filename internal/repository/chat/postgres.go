package chat

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opGetOrCreateChat = "db.chat.GetOrCreateChat"
	opCreateMessage   = "db.chat.CreateMessage"
	opUpdateAdStatus  = "db.chat.UpdateAdStatus"
)

// PgxPool интерфейс для пула соединений (или транзакции),
// позволяющий подменять его моком в тестах.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
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

	// Сначала пытаемся получить существующий чат
	const querySelect = `
		SELECT id FROM chat
		WHERE product_id = $1 AND buyer_id = $2 AND seller_id = $3
		LIMIT 1
	`

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

	// Если чата нет, создаем новый
	const queryInsert = `
		INSERT INTO chat (product_id, buyer_id, seller_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`

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

// UpdateAdStatus обновляет статус объявления.
func (cs *ChatStorage) UpdateAdStatus(
	ctx context.Context,
	adID int64,
	status string,
) error {
	const query = `
		UPDATE product
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	commandTag, err := cs.pool.Exec(ctx, query, status, adID)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to update ad status",
			slog.String("op", opUpdateAdStatus),
			slog.String("error", err.Error()),
		)
		return err
	}

	if commandTag.RowsAffected() == 0 {
		cs.log.WarnContext(ctx, "ad not found when updating status",
			slog.String("op", opUpdateAdStatus),
			slog.Int64("ad_id", adID),
		)
		return errors.New("ad not found")
	}

	return nil
}
