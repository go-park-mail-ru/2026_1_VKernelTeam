package chat

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatStorage_GetOrCreateChat(t *testing.T) {
	ctx := context.Background()

	t.Run("Existing chat returned", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(77)))

		id, err := storage.GetOrCreateChat(ctx, 38, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, int64(77), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("New chat created when none exists", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(88)))

		id, err := storage.GetOrCreateChat(ctx, 38, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, int64(88), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Select returns non-nil error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnError(errors.New("db down"))

		_, err = storage.GetOrCreateChat(ctx, 38, 1, 2)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Insert error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO chat")).
			WithArgs(int64(38), int64(1), int64(2)).
			WillReturnError(errors.New("constraint violation"))

		_, err = storage.GetOrCreateChat(ctx, 38, 1, 2)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChatStorage_CreateMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO message")).
			WithArgs(int64(77), int64(1), "hi", "order").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(555)))

		id, err := storage.CreateMessage(ctx, &models.Message{
			ChatID: 77, SenderID: 1, Text: "hi", Type: "order",
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(555), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO message")).
			WithArgs(int64(77), int64(1), "hi", "order").
			WillReturnError(errors.New("db err"))

		_, err = storage.CreateMessage(ctx, &models.Message{
			ChatID: 77, SenderID: 1, Text: "hi", Type: "order",
		})
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChatStorage_GetChatByID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		rows := pgxmock.NewRows([]string{"id", "product_id", "buyer_id", "seller_id", "created_at", "updated_at"}).
			AddRow(int64(5), int64(38), int64(1), int64(2), now, now)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, product_id, buyer_id, seller_id")).
			WithArgs(int64(5)).
			WillReturnRows(rows)

		chat, err := storage.GetChatByID(ctx, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), chat.ID)
		assert.Equal(t, int64(38), chat.AdID)
		assert.Equal(t, int64(1), chat.BuyerID)
		assert.Equal(t, int64(2), chat.SellerID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, product_id, buyer_id, seller_id")).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		_, err = storage.GetChatByID(ctx, 99)
		assert.ErrorIs(t, err, ErrChatNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Other error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, product_id, buyer_id, seller_id")).
			WithArgs(int64(5)).
			WillReturnError(errors.New("db down"))

		_, err = storage.GetChatByID(ctx, 5)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrChatNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChatStorage_CompletePurchase(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "order"`)).
			WithArgs(int64(1), int64(1000)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_item")).
			WithArgs(int64(101), int64(38), int64(1000)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status = 'sold'")).
			WithArgs(int64(38)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE product_id")).
			WithArgs(int64(38)).
			WillReturnResult(pgxmock.NewResult("DELETE", 3))
		mock.ExpectCommit()

		err = storage.CompletePurchase(ctx, 1, 38, 1000)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Begin error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		err = storage.CompletePurchase(ctx, 1, 38, 1000)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Create order error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "order"`)).
			WithArgs(int64(1), int64(1000)).
			WillReturnError(errors.New("pk violation"))
		mock.ExpectRollback()

		err = storage.CompletePurchase(ctx, 1, 38, 1000)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ad not found on update", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "order"`)).
			WithArgs(int64(1), int64(1000)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_item")).
			WithArgs(int64(101), int64(38), int64(1000)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status = 'sold'")).
			WithArgs(int64(38)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectRollback()

		err = storage.CompletePurchase(ctx, 1, 38, 1000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ad not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Clear carts error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "order"`)).
			WithArgs(int64(1), int64(1000)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_item")).
			WithArgs(int64(101), int64(38), int64(1000)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status = 'sold'")).
			WithArgs(int64(38)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE product_id")).
			WithArgs(int64(38)).
			WillReturnError(errors.New("lock"))
		mock.ExpectRollback()

		err = storage.CompletePurchase(ctx, 1, 38, 1000)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChatStorage_GetChatsByUserID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	chatsColumns := []string{
		"chat_id",
		"ad_id", "title", "price", "status", "ad_photo",
		"partner_id", "partner_name", "partner_avatar",
		"last_text", "last_type", "last_at",
	}

	t.Run("Success with last message", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		rows := pgxmock.NewRows(chatsColumns).
			AddRow(
				int64(7),
				int64(38), "iPhone", int64(1000), "active", "photo.jpg",
				int64(2), "Иван", "avatar.png",
				"hello", "text", now,
			)

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		result, err := storage.GetChatsByUserID(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(7), result[0].ChatID)
		assert.Equal(t, "iPhone", result[0].Ad.Title)
		assert.NotNil(t, result[0].LastMessage)
		assert.Equal(t, "hello", result[0].LastMessage.Text)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success without last message", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		rows := pgxmock.NewRows(chatsColumns).
			AddRow(
				int64(7),
				int64(38), "iPhone", int64(1000), "active", "",
				int64(2), "Иван", "",
				nil, nil, nil,
			)

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		result, err := storage.GetChatsByUserID(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Nil(t, result[0].LastMessage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Empty list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows(chatsColumns))

		result, err := storage.GetChatsByUserID(ctx, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db down"))

		_, err = storage.GetChatsByUserID(ctx, 1)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChatStorage_GetChatDetail(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	headerColumns := []string{
		"chat_id",
		"ad_id", "title", "price", "status", "ad_photo",
		"partner_id", "partner_name", "partner_avatar",
	}
	messagesColumns := []string{"id", "sender_id", "text_content", "msg_type", "created_at"}

	t.Run("Success with messages", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(5), int64(1)).
			WillReturnRows(pgxmock.NewRows(headerColumns).AddRow(
				int64(5),
				int64(38), "iPhone", int64(1000), "active", "photo.jpg",
				int64(2), "Иван", "avatar.png",
			))
		mock.ExpectQuery(regexp.QuoteMeta("FROM message")).
			WithArgs(int64(5)).
			WillReturnRows(pgxmock.NewRows(messagesColumns).
				AddRow(int64(1), int64(1), "hi", "order", now).
				AddRow(int64(2), int64(2), "ok", "text", now.Add(time.Minute)))

		detail, err := storage.GetChatDetail(ctx, 5, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(5), detail.ChatID)
		assert.Equal(t, "iPhone", detail.Ad.Title)
		assert.Equal(t, int64(2), detail.Partner.ID)
		assert.Len(t, detail.Messages, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Not found or not a participant", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(5), int64(99)).
			WillReturnError(pgx.ErrNoRows)

		_, err = storage.GetChatDetail(ctx, 5, 99)
		assert.ErrorIs(t, err, ErrChatNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Header query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(5), int64(1)).
			WillReturnError(errors.New("db down"))

		_, err = storage.GetChatDetail(ctx, 5, 1)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrChatNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Messages query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(5), int64(1)).
			WillReturnRows(pgxmock.NewRows(headerColumns).AddRow(
				int64(5),
				int64(38), "iPhone", int64(1000), "active", "",
				int64(2), "Иван", "",
			))
		mock.ExpectQuery(regexp.QuoteMeta("FROM message")).
			WithArgs(int64(5)).
			WillReturnError(errors.New("db down"))

		_, err = storage.GetChatDetail(ctx, 5, 1)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("No messages", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewChatStorage(mock, slog.Default())

		mock.ExpectQuery(regexp.QuoteMeta("FROM chat c")).
			WithArgs(int64(5), int64(1)).
			WillReturnRows(pgxmock.NewRows(headerColumns).AddRow(
				int64(5),
				int64(38), "iPhone", int64(1000), "active", "",
				int64(2), "Иван", "",
			))
		mock.ExpectQuery(regexp.QuoteMeta("FROM message")).
			WithArgs(int64(5)).
			WillReturnRows(pgxmock.NewRows(messagesColumns))

		detail, err := storage.GetChatDetail(ctx, 5, 1)
		require.NoError(t, err)
		assert.NotNil(t, detail.Messages)
		assert.Len(t, detail.Messages, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
