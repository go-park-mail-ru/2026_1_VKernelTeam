package cart

import (
	"context"
	"fmt"
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

	storage := NewCartStorage(mock)
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

	storage := NewCartStorage(mock)
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

	storage := NewCartStorage(mock)
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

	storage := NewCartStorage(mock)
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

func TestCartStorage_Checkout(t *testing.T) {
	ctx := context.Background()

	cartColumns := []string{"id", "seller_id", "price", "status", "user_id", "first_name", "email"}

	t.Run("Success single seller", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		cartRows := pgxmock.NewRows(cartColumns).
			AddRow(int64(10), int64(2), int64(5000), "active", int64(2), "Иван", "ivan@mail.ru")

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnRows(cartRows)

		// create order
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"order\"")).
			WithArgs(int64(1), int64(5000)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))

		// create order item
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_item")).
			WithArgs(int64(101), int64(10), int64(5000)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// update product status
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status")).
			WithArgs(int64(10)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		// clear cart
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE user_id")).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		mock.ExpectCommit()

		orderIDs, sellers, err := storage.Checkout(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, orderIDs, 1)
		assert.Contains(t, orderIDs, int64(101))
		assert.Len(t, sellers, 1)
		assert.Equal(t, "Иван", sellers[2].Name)
		assert.Equal(t, "ivan@mail.ru", sellers[2].Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Empty cart", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows(cartColumns))
		mock.ExpectRollback()

		orderIDs, sellers, err := storage.Checkout(ctx, 1)
		assert.ErrorIs(t, err, ErrCartEmpty)
		assert.Nil(t, orderIDs)
		assert.Nil(t, sellers)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Product reserved", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		cartRows := pgxmock.NewRows(cartColumns).
			AddRow(int64(10), int64(2), int64(5000), "reserved", int64(2), "Иван", "ivan@mail.ru")

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnRows(cartRows)
		mock.ExpectRollback()

		_, _, err = storage.Checkout(ctx, 1)
		assert.ErrorIs(t, err, ErrProductReserved)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Begin tx error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("db unavailable"))

		_, _, err = storage.Checkout(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "begin tx")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query cart items error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		_, _, err = storage.Checkout(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query cart items")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Create order error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		cartRows := pgxmock.NewRows(cartColumns).
			AddRow(int64(10), int64(2), int64(5000), "active", int64(2), "Иван", "ivan@mail.ru")

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnRows(cartRows)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"order\"")).
			WithArgs(int64(1), int64(5000)).
			WillReturnError(fmt.Errorf("constraint violation"))
		mock.ExpectRollback()

		_, _, err = storage.Checkout(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create order")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Clear cart error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		storage := NewCartStorage(mock)

		cartRows := pgxmock.NewRows(cartColumns).
			AddRow(int64(10), int64(2), int64(5000), "active", int64(2), "Иван", "ivan@mail.ru")

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.id")).
			WithArgs(int64(1)).
			WillReturnRows(cartRows)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"order\"")).
			WithArgs(int64(1), int64(5000)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))

		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_item")).
			WithArgs(int64(101), int64(10), int64(5000)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status")).
			WithArgs(int64(10)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM cart_item WHERE user_id")).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		_, _, err = storage.Checkout(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "clear cart")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
