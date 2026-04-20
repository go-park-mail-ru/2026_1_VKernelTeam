package chat

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// const (
// 	opGetOrCreateChat = "db.chat.GetOrCreateChat"
// 	opCreateMessage   = "db.chat.CreateMessage"
// 	opUpdateAdStatus  = "db.chat.UpdateAdStatus"
// )

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
	// TODO: Реализовать логику получения или создания чата
	return 0, nil
}

// CreateMessage создает новое сообщение в чате.
func (cs *ChatStorage) CreateMessage(
	ctx context.Context,
	message *models.Message,
) (int64, error) {
	// TODO: Реализовать создание сообщения в БД
	return 0, nil
}

// UpdateAdStatus обновляет статус объявления.
func (cs *ChatStorage) UpdateAdStatus(
	ctx context.Context,
	adID int64,
	status string,
) error {
	// TODO: Реализовать обновление статуса объявления
	return nil
}
