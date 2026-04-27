package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opGetOrCreateChat  = "db.chat.GetOrCreateChat"
	opCreateMessage    = "db.chat.CreateMessage"
	opGetChatByID      = "db.chat.GetChatByID"
	opCompletePurchase = "db.chat.CompletePurchase"
	opGetChatsByUserID = "db.chat.GetChatsByUserID"
	opGetChatDetail    = "db.chat.GetChatDetail"
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
		INSERT INTO message (chat_id, sender_id, text_content, msg_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
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

// GetChatsByUserID возвращает список чатов (с превью объявления, собеседника
// и последнего сообщения), в которых участвует пользователь. Сортировка — по
// времени последнего сообщения (пусто - внизу), далее по дате создания чата.
func (cs *ChatStorage) GetChatsByUserID(ctx context.Context, userID int64) ([]dto.ChatPreview, error) {
	const query = `
		SELECT
			c.id,
			p.id, p.title, p.price, p.status,
			COALESCE(
				(SELECT file_path FROM product_image
				 WHERE product_id = p.id ORDER BY sort_order ASC LIMIT 1),
				''
			) AS ad_photo,
			CASE WHEN c.buyer_id = $1 THEN c.seller_id ELSE c.buyer_id END AS partner_id,
			u.first_name,
			COALESCE(u.avatar_path, '') AS avatar_path,
			m.text_content, m.msg_type, m.created_at
		FROM chat c
		JOIN product p ON p.id = c.product_id
		JOIN "user" u ON u.id = CASE WHEN c.buyer_id = $1 THEN c.seller_id ELSE c.buyer_id END
		LEFT JOIN LATERAL (
			SELECT text_content, msg_type, created_at
			FROM message
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON TRUE
		WHERE c.buyer_id = $1 OR c.seller_id = $1
		ORDER BY m.created_at DESC NULLS LAST, c.created_at DESC
	`

	rows, err := cs.pool.Query(ctx, query, userID)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to query user chats",
			slog.String("op", opGetChatsByUserID),
			slog.String("user_id", fmt.Sprint(userID)),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("GetChatsByUserID: query: %w", err)
	}
	defer rows.Close()

	var result []dto.ChatPreview
	for rows.Next() {
		var (
			preview  dto.ChatPreview
			lastText sql.NullString
			lastType sql.NullString
			lastAt   sql.NullTime
		)
		if err := rows.Scan(
			&preview.ChatID,
			&preview.Ad.ID, &preview.Ad.Title, &preview.Ad.Price, &preview.Ad.Status,
			&preview.Ad.Photo,
			&preview.Partner.ID, &preview.Partner.Name, &preview.Partner.AvatarPath,
			&lastText, &lastType, &lastAt,
		); err != nil {
			cs.log.ErrorContext(ctx, "failed to scan chat preview",
				slog.String("op", opGetChatsByUserID),
				slog.String("user_id", fmt.Sprint(userID)),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("GetChatsByUserID: scan: %w", err)
		}

		// Если есть последнее сообщение, добавляем его в превью
		if lastAt.Valid {
			preview.LastMessage = &dto.LastMessagePreview{
				Text:      lastText.String,
				Type:      lastType.String,
				CreatedAt: lastAt.Time,
			}
		}
		result = append(result, preview)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetChatsByUserID: rows: %w", err)
	}

	if result == nil {
		result = []dto.ChatPreview{}
	}
	return result, nil
}

// GetChatDetail возвращает шапку чата (объявление + собеседник) и все его
// сообщения. Доступ есть только у участников; чужому или несуществующему чату
// отдаёт ErrChatNotFound.
func (cs *ChatStorage) GetChatDetail(ctx context.Context, chatID, userID int64) (dto.ChatDetailResponse, error) {
	const headerQuery = `
		SELECT
			c.id,
			p.id, p.title, p.price, p.status,
			COALESCE(
				(SELECT file_path FROM product_image
				 WHERE product_id = p.id ORDER BY sort_order ASC LIMIT 1),
				''
			) AS ad_photo,
			CASE WHEN c.buyer_id = $2 THEN c.seller_id ELSE c.buyer_id END AS partner_id,
			u.first_name,
			COALESCE(u.avatar_path, '') AS avatar_path
		FROM chat c
		JOIN product p ON p.id = c.product_id
		JOIN "user" u ON u.id = CASE WHEN c.buyer_id = $2 THEN c.seller_id ELSE c.buyer_id END
		WHERE c.id = $1 AND (c.buyer_id = $2 OR c.seller_id = $2)
	`

	var resp dto.ChatDetailResponse
	err := cs.pool.QueryRow(ctx, headerQuery, chatID, userID).Scan(
		&resp.ChatID,
		&resp.Ad.ID, &resp.Ad.Title, &resp.Ad.Price, &resp.Ad.Status,
		&resp.Ad.Photo,
		&resp.Partner.ID, &resp.Partner.Name, &resp.Partner.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Либо чата нет, либо юзер не участник
			return dto.ChatDetailResponse{}, ErrChatNotFound
		}
		cs.log.ErrorContext(ctx, "failed to get chat header",
			slog.String("op", opGetChatDetail),
			slog.String("error", err.Error()),
			slog.String("chat_id", fmt.Sprint(chatID)),
			slog.String("user_id", fmt.Sprint(userID)),
		)
		return dto.ChatDetailResponse{}, fmt.Errorf("GetChatDetail: header: %w", err)
	}

	const messagesQuery = `
		SELECT id, sender_id, text_content, msg_type, created_at
		FROM message
		WHERE chat_id = $1
		ORDER BY created_at ASC
	`

	// Получаем все сообщения чата
	rows, err := cs.pool.Query(ctx, messagesQuery, chatID)
	if err != nil {
		cs.log.ErrorContext(ctx, "failed to query messages",
			slog.String("op", opGetChatDetail),
			slog.String("chat_id", fmt.Sprint(chatID)),
			slog.String("user_id", fmt.Sprint(userID)),
			slog.String("error", err.Error()),
		)
		return dto.ChatDetailResponse{}, fmt.Errorf("GetChatDetail: messages query: %w", err)
	}
	defer rows.Close()

	resp.Messages = []dto.MessageItem{}
	for rows.Next() {
		var m dto.MessageItem
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Text, &m.Type, &m.CreatedAt); err != nil {
			return dto.ChatDetailResponse{}, fmt.Errorf("GetChatDetail: scan message: %w", err)
		}
		resp.Messages = append(resp.Messages, m)
	}
	if err := rows.Err(); err != nil {
		return dto.ChatDetailResponse{}, fmt.Errorf("GetChatDetail: messages rows: %w", err)
	}

	return resp, nil
}
