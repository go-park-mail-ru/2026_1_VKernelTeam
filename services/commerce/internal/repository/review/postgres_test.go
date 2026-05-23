package review

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	colCreatedAt    = "created_at"
	colUpdatedAt    = "updated_at"
	colRating       = "rating"
	colCount        = "count"
	testContentGood = "great seller"
)

func newStorage(t *testing.T) (*ReviewStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return NewStorage(mock, slog.Default()), mock
}

func TestReviewStorage_Create(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO review")).
			WithArgs(int64(1), int64(2), int64(38), 5, testContentGood).
			WillReturnRows(pgxmock.NewRows([]string{"id", colCreatedAt, colUpdatedAt}).
				AddRow(int64(101), now, now))

		got, err := s.Create(ctx, &models.Review{
			SenderID: 1, ReceiverID: 2, ProductID: 38, Rating: 5, Content: testContentGood,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(101), got.ID)
		assert.Equal(t, 5, got.Rating)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already exists on 23505", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO review")).
			WithArgs(int64(1), int64(2), int64(38), 5, testContentGood).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		_, err := s.Create(ctx, &models.Review{
			SenderID: 1, ReceiverID: 2, ProductID: 38, Rating: 5, Content: testContentGood,
		})
		assert.ErrorIs(t, err, ErrReviewAlreadyExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other pg error", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO review")).
			WithArgs(int64(1), int64(2), int64(38), 5, testContentGood).
			WillReturnError(&pgconn.PgError{Code: "23514"})

		_, err := s.Create(ctx, &models.Review{
			SenderID: 1, ReceiverID: 2, ProductID: 38, Rating: 5, Content: testContentGood,
		})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrReviewAlreadyExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_Update(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("UPDATE review")).
			WithArgs(int64(101), int64(1), 4, "ok").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "sender_id", "receiver_id", "product_id", colRating, "content", colCreatedAt, colUpdatedAt,
			}).AddRow(int64(101), int64(1), int64(2), int64(38), 4, "ok", now, now))

		r, err := s.Update(ctx, 101, 1, 4, "ok")
		require.NoError(t, err)
		assert.Equal(t, 4, r.Rating)
		assert.Equal(t, "ok", r.Content)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("UPDATE review")).
			WithArgs(int64(101), int64(1), 4, "ok").
			WillReturnError(pgx.ErrNoRows)

		_, err := s.Update(ctx, 101, 1, 4, "ok")
		assert.ErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other error", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta("UPDATE review")).
			WithArgs(int64(101), int64(1), 4, "ok").
			WillReturnError(errors.New("db down"))

		_, err := s.Update(ctx, 101, 1, 4, "ok")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM review")).
			WithArgs(int64(101), int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := s.Delete(ctx, 101, 1)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM review")).
			WithArgs(int64(101), int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := s.Delete(ctx, 101, 1)
		assert.ErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM review")).
			WithArgs(int64(101), int64(1)).
			WillReturnError(errors.New("db down"))

		err := s.Delete(ctx, 101, 1)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_GetByID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, sender_id, receiver_id, product_id, rating, content")).
			WithArgs(int64(101)).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "sender_id", "receiver_id", "product_id", colRating, "content", colCreatedAt, colUpdatedAt,
			}).AddRow(int64(101), int64(1), int64(2), int64(38), 5, "ok", now, now))

		r, err := s.GetByID(ctx, 101)
		require.NoError(t, err)
		assert.Equal(t, int64(1), r.SenderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, sender_id, receiver_id, product_id, rating, content")).
			WithArgs(int64(101)).
			WillReturnError(pgx.ErrNoRows)

		_, err := s.GetByID(ctx, 101)
		assert.ErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, sender_id, receiver_id, product_id, rating, content")).
			WithArgs(int64(101)).
			WillReturnError(errors.New("db down"))

		_, err := s.GetByID(ctx, 101)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_GetResponseByID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	// Те же колонки, что и у списочных методов (без cursor-полей).
	cols := []string{
		"r_id", "r_receiver_id", "r_rating", "r_content", "r_created_at", "r_updated_at",
		"u_id", "u_first_name", "u_avatar_path",
		"p_id", "p_title", "p_price", "p_status", "photo",
	}

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)
		rows := pgxmock.NewRows(cols).
			AddRow(int64(101), int64(2), 5, testContentGood, now, now,
				int64(1), "Ivan", "ava.png",
				int64(38), "iPhone", int64(1000), "active", "img.jpg")

		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(101)).
			WillReturnRows(rows)

		got, err := s.GetResponseByID(ctx, 101)
		require.NoError(t, err)
		assert.Equal(t, int64(101), got.ID)
		assert.Equal(t, int64(2), got.ReceiverID)
		assert.Equal(t, 5, got.Rating)
		assert.Equal(t, testContentGood, got.Content)
		assert.Equal(t, int64(1), got.Sender.ID)
		assert.Equal(t, "Ivan", got.Sender.Name)
		assert.Equal(t, "ava.png", got.Sender.AvatarPath)
		assert.Equal(t, int64(38), got.Product.ID)
		assert.Equal(t, "iPhone", got.Product.Title)
		assert.Equal(t, int64(1000), got.Product.Price)
		assert.Equal(t, "active", got.Product.Status)
		assert.Equal(t, "img.jpg", got.Product.Photo)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(101)).
			WillReturnError(pgx.ErrNoRows)

		_, err := s.GetResponseByID(ctx, 101)
		assert.ErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(101)).
			WillReturnError(errors.New("db down"))

		_, err := s.GetResponseByID(ctx, 101)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrReviewNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func listColumns() []string {
	return []string{
		"r_id", "r_receiver_id", "r_rating", "r_content", "r_created_at", "r_updated_at",
		"u_id", "u_first_name", "u_avatar_path",
		"p_id", "p_title", "p_price", "p_status", "photo",
	}
}

func TestReviewStorage_ListByReceiver(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("no cursor, empty", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(2), (*int64)(nil), 21).
			WillReturnRows(pgxmock.NewRows(listColumns()))

		out, err := s.ListByReceiver(ctx, 2, nil, 21)
		require.NoError(t, err)
		assert.Len(t, out, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with cursor and rows", func(t *testing.T) {
		s, mock := newStorage(t)
		cur := int64(500)
		rows := pgxmock.NewRows(listColumns()).
			AddRow(int64(499), int64(2), 5, "ok", now, now,
				int64(1), "Ivan", "ava.png",
				int64(38), "iPhone", int64(1000), "active", "img.jpg")

		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(2), &cur, 5).
			WillReturnRows(rows)

		out, err := s.ListByReceiver(ctx, 2, &cur, 5)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, int64(499), out[0].ID)
		assert.Equal(t, "Ivan", out[0].Sender.Name)
		assert.Equal(t, "img.jpg", out[0].Product.Photo)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(2), (*int64)(nil), 5).
			WillReturnError(errors.New("db down"))

		_, err := s.ListByReceiver(ctx, 2, nil, 5)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_ListBySender(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("no cursor, single row", func(t *testing.T) {
		s, mock := newStorage(t)
		rows := pgxmock.NewRows(listColumns()).
			AddRow(int64(101), int64(2), 4, "fine", now, now,
				int64(1), "Ivan", "",
				int64(38), "iPhone", int64(1000), "active", "")

		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(1), (*int64)(nil), 21).
			WillReturnRows(rows)

		out, err := s.ListBySender(ctx, 1, nil, 21)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, int64(2), out[0].ReceiverID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with cursor empty", func(t *testing.T) {
		s, mock := newStorage(t)
		cur := int64(100)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(1), &cur, 21).
			WillReturnRows(pgxmock.NewRows(listColumns()))

		out, err := s.ListBySender(ctx, 1, &cur, 21)
		require.NoError(t, err)
		assert.Len(t, out, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("FROM review r")).
			WithArgs(int64(1), (*int64)(nil), 5).
			WillReturnError(errors.New("db down"))

		_, err := s.ListBySender(ctx, 1, nil, 5)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_SummaryByReceiver(t *testing.T) {
	ctx := context.Background()

	t.Run("success with distribution and user", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, COUNT(*)`)).
			WithArgs(int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{colRating, colCount}).
				AddRow(5, 3).AddRow(4, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, reviews_count FROM "user"`)).
			WithArgs(int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{colRating, "reviews_count"}).
				AddRow(4.75, 4))

		got, err := s.SummaryByReceiver(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, 4, got.Total)
		assert.InDelta(t, 4.75, got.Average, 0.001)
		assert.Equal(t, 3, got.Distribution[5])
		assert.Equal(t, 1, got.Distribution[4])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found returns zeros", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, COUNT(*)`)).
			WithArgs(int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{colRating, colCount}))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, reviews_count FROM "user"`)).
			WithArgs(int64(2)).
			WillReturnError(pgx.ErrNoRows)

		got, err := s.SummaryByReceiver(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, 0, got.Total)
		assert.Equal(t, float64(0), got.Average)
		assert.Empty(t, got.Distribution)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("distribution query error", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, COUNT(*)`)).
			WithArgs(int64(2)).
			WillReturnError(errors.New("db down"))

		_, err := s.SummaryByReceiver(ctx, 2)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user aggregates other error", func(t *testing.T) {
		s, mock := newStorage(t)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, COUNT(*)`)).
			WithArgs(int64(2)).
			WillReturnRows(pgxmock.NewRows([]string{colRating, colCount}).AddRow(5, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT rating, reviews_count FROM "user"`)).
			WithArgs(int64(2)).
			WillReturnError(errors.New("db down"))

		_, err := s.SummaryByReceiver(ctx, 2)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_ExistsPurchaseRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("true", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
			WithArgs(int64(1), int64(2), int64(38)).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		ok, err := s.ExistsPurchaseRequest(ctx, 1, 2, 38)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("false", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
			WithArgs(int64(1), int64(2), int64(38)).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		ok, err := s.ExistsPurchaseRequest(ctx, 1, 2, 38)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
			WithArgs(int64(1), int64(2), int64(38)).
			WillReturnError(errors.New("db down"))

		_, err := s.ExistsPurchaseRequest(ctx, 1, 2, 38)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReviewStorage_GetProductSellerID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT seller_id FROM product")).
			WithArgs(int64(38)).
			WillReturnRows(pgxmock.NewRows([]string{"seller_id"}).AddRow(int64(2)))

		sellerID, err := s.GetProductSellerID(ctx, 38)
		require.NoError(t, err)
		assert.Equal(t, int64(2), sellerID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT seller_id FROM product")).
			WithArgs(int64(38)).
			WillReturnError(pgx.ErrNoRows)

		_, err := s.GetProductSellerID(ctx, 38)
		assert.ErrorIs(t, err, ErrProductNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		s, mock := newStorage(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT seller_id FROM product")).
			WithArgs(int64(38)).
			WillReturnError(errors.New("db down"))

		_, err := s.GetProductSellerID(ctx, 38)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrProductNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
