package cart

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartStorage_Add(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewCartStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart_item")).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := storage.Add(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Product already in cart", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart_item")).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := storage.Add(ctx, 1, 10)
		assert.ErrorIs(t, err, ErrProductAlreadyInCart)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exec error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart_item")).
			WithArgs(int64(1), int64(10)).
			WillReturnError(fmt.Errorf("connection refused"))

		err := storage.Add(ctx, 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CartStorage.Add")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCartStorage_Remove(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewCartStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item")).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := storage.Remove(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Item not found", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item")).
			WithArgs(int64(1), int64(99)).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := storage.Remove(ctx, 1, 99)
		assert.ErrorIs(t, err, ErrCartItemNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exec error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item")).
			WithArgs(int64(1), int64(10)).
			WillReturnError(assert.AnError)

		err := storage.Remove(ctx, 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CartStorage.Remove")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCartStorage_GetByUserID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewCartStorage(mock, slog.Default())
	ctx := context.Background()

	columns := []string{"id", "title", "price", "seller_id", "first_name", "image_path"}

	t.Run("Success with items", func(t *testing.T) {
		imgPath := "/img/phone.jpg"
		rows := pgxmock.NewRows(columns).
			AddRow(int64(10), "iPhone", int64(100000), int64(2), "Иван", &imgPath).
			AddRow(int64(20), "MacBook", int64(200000), int64(3), "Петр", (*string)(nil))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		items, err := storage.GetByUserID(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, items, 2)

		assert.Equal(t, int64(10), items[0].ProductID)
		assert.Equal(t, "iPhone", items[0].Title)
		assert.Equal(t, int64(100000), items[0].Price)
		assert.Equal(t, "/img/phone.jpg", items[0].ImagePath)

		assert.Equal(t, int64(20), items[1].ProductID)
		assert.Equal(t, "", items[1].ImagePath) // nil image_path -> ""
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Empty cart returns empty slice", func(t *testing.T) {
		rows := pgxmock.NewRows(columns)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		items, err := storage.GetByUserID(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, items)
		assert.Len(t, items, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		items, err := storage.GetByUserID(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "CartStorage.GetByUserID")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCartStorage_Clear(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewCartStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE user_id")).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 3))

		err := storage.Clear(ctx, 1)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success empty cart", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE user_id")).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := storage.Clear(ctx, 1)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exec error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE user_id")).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		err := storage.Clear(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CartStorage.Clear")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

