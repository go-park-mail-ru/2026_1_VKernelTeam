// Package system_message предоставляет отправку системных сообщений в чат
// от имени системного пользователя. Пишет напрямую в таблицы chat/message
// (БД у всех сервисов одна), что позволяет избежать межсервисного gRPC
// до тех пор, пока у commerce нет собственного gRPC-сервера.
package system_message

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// messageTypeSystem дублирует константу из commerce/internal/domain/models —
// импортировать internal-пакет другого сервиса нельзя.
const messageTypeSystem = "system"

const (
	opSend = "db.system_message.Send"
)

// PgxPool — минимальный интерфейс пула pgx.
type PgxPool interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// Storage отвечает за запись системных сообщений в чат.
type Storage struct {
	pool PgxPool
	log  *slog.Logger
}

// New создаёт хранилище.
func New(pool PgxPool, log *slog.Logger) *Storage {
	return &Storage{pool: pool, log: log}
}

// Send отправляет системное сообщение получателю в чат, связанный с adID.
// systemUserID — id системного пользователя (хранится в platform_setting).
// Чат создаётся (или переиспользуется) с buyer_id=systemUserID, seller_id=recipientID.
func (s *Storage) Send(
	ctx context.Context,
	systemUserID, recipientID, adID int64,
	text string,
) error {
	if adID <= 0 {
		return errors.New("system_message.Send: ad_id required")
	}

	var chatID int64
	const upsertChatQ = `
		INSERT INTO chat (product_id, buyer_id, seller_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (product_id, buyer_id, seller_id) DO UPDATE
		    SET updated_at = NOW()
		RETURNING id
	`
	if err := s.pool.QueryRow(ctx, upsertChatQ, adID, systemUserID, recipientID).Scan(&chatID); err != nil {
		s.log.ErrorContext(ctx, "failed to upsert system chat",
			slog.String("op", opSend),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("system_message.Send: upsert chat: %w", err)
	}

	const insertMsgQ = `
		INSERT INTO message (chat_id, sender_id, text_content, msg_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	if _, err := s.pool.Exec(ctx, insertMsgQ, chatID, systemUserID, text, messageTypeSystem); err != nil {
		s.log.ErrorContext(ctx, "failed to insert system message",
			slog.String("op", opSend),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("system_message.Send: insert message: %w", err)
	}

	s.log.InfoContext(ctx, "system message sent",
		slog.String("op", opSend),
		slog.Int64("ad_id", adID),
		slog.Int64("recipient_id", recipientID),
	)
	return nil
}
