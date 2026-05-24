package purchase

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStorage(t *testing.T) (*PurchaseStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return NewStorage(mock, slog.Default()), mock
}

func listColumns() []string {
	return []string{
		"order_id", "product_id",
		"title", "price",
		"photo",
		"location",
		"seller_id", "seller_name", "seller_avatar",
		"source", "purchased_at", "chat_id",
	}
}

func TestPurchaseStorage_ListByBuyer(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("no cursor, empty", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "order" o`)).
			WithArgs(int64(1), (*int64)(nil), 21).
			WillReturnRows(pgxmock.NewRows(listColumns()))

		out, err := s.ListByBuyer(ctx, 1, nil, 21)
		require.NoError(t, err)
		assert.Len(t, out, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with cursor and rows — chat source", func(t *testing.T) {
		s, mock := newStorage(t)
		cur := int64(500)
		chatID := int64(78)
		rows := pgxmock.NewRows(listColumns()).
			AddRow(int64(499), int64(38),
				"iPhone 13 mini", int64(35000),
				"main.jpg",
				"Москва",
				int64(42), "Иван", "ava.png",
				"chat", now, &chatID)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "order" o`)).
			WithArgs(int64(1), &cur, 5).
			WillReturnRows(rows)

		out, err := s.ListByBuyer(ctx, 1, &cur, 5)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, int64(499), out[0].OrderID)
		assert.Equal(t, int64(38), out[0].ProductID)
		assert.Equal(t, "iPhone 13 mini", out[0].Title)
		assert.Equal(t, int64(35000), out[0].Price)
		assert.Equal(t, "main.jpg", out[0].Photo)
		assert.Equal(t, "Москва", out[0].Location)
		assert.Equal(t, int64(42), out[0].Seller.ID)
		assert.Equal(t, "Иван", out[0].Seller.Name)
		assert.Equal(t, "chat", out[0].Source)
		require.NotNil(t, out[0].ChatID)
		assert.Equal(t, int64(78), *out[0].ChatID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cart source — chat_id null", func(t *testing.T) {
		s, mock := newStorage(t)
		rows := pgxmock.NewRows(listColumns()).
			AddRow(int64(100), int64(7),
				"Книга", int64(500),
				"",
				"",
				int64(11), "Анна", "",
				"cart", now, (*int64)(nil))

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "order" o`)).
			WithArgs(int64(1), (*int64)(nil), 21).
			WillReturnRows(rows)

		out, err := s.ListByBuyer(ctx, 1, nil, 21)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "cart", out[0].Source)
		assert.Nil(t, out[0].ChatID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "order" o`)).
			WithArgs(int64(1), (*int64)(nil), 5).
			WillReturnError(errors.New("db down"))

		_, err := s.ListByBuyer(ctx, 1, nil, 5)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error — wrong column types", func(t *testing.T) {
		s, mock := newStorage(t)
		rows := pgxmock.NewRows([]string{"order_id"}).AddRow("not-an-int")
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "order" o`)).
			WithArgs(int64(1), (*int64)(nil), 5).
			WillReturnRows(rows)

		_, err := s.ListByBuyer(ctx, 1, nil, 5)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
