package ad

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestAdStorage_GetAdByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	adID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "seller_id", "category_id", "title", "description", "price", "status", "location", "created_at", "updated_at", "photos", "views_count", "favorites_count",
		}).AddRow(adID, int64(2), int64(3), "Title", "Desc", int64(100), "active", "Loc", now, now, []string{"p1.jpg"}, int64(10), int64(5))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(adID).
			WillReturnRows(rows)

		ad, err := storage.GetAdByID(ctx, adID)
		assert.NoError(t, err)
		assert.Equal(t, adID, ad.ID)
		assert.Equal(t, []string{"p1.jpg"}, ad.Photos)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(adID).
			WillReturnError(pgx.ErrNoRows)

		_, err := storage.GetAdByID(ctx, adID)
		assert.ErrorIs(t, err, ErrAdNotFound)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(adID).
			WillReturnError(assert.AnError)

		_, err := storage.GetAdByID(ctx, adID)
		assert.Error(t, err)
	})
}

func TestAdStorage_GetAllAds(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "seller_id", "category_id", "title", "description", "price", "status", "location", "created_at", "updated_at", "photos", "views_count", "favorites_count",
		}).AddRow(int64(1), int64(2), int64(3), "Title", "Desc", int64(100), "active", "Loc", now, now, []string{"p1.jpg"}, int64(10), int64(5))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnRows(rows)

		ads, err := storage.GetAllAds(ctx)
		assert.NoError(t, err)
		assert.Len(t, ads, 1)
		assert.Equal(t, int64(1), ads[0].ID)
	})

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnRows(pgxmock.NewRows([]string{"id"}))

		ads, err := storage.GetAllAds(ctx)
		assert.NoError(t, err)
		assert.Len(t, ads, 0)
		assert.NotNil(t, ads)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnError(assert.AnError)

		_, err := storage.GetAllAds(ctx)
		assert.Error(t, err)
	})
}

func TestAdStorage_CreateAd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	req := &dto.CreateAdRequest{
		UserID:      1,
		CategoryID:  2,
		Title:       "Title",
		Description: "Desc",
		Price:       100,
		Status:      "active",
		Location:    "Loc",
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO product")).
			WithArgs(req.UserID, req.CategoryID, req.Title, req.Description, req.Price, req.Status, req.Location).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(123)))

		id, err := storage.CreateAd(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int64(123), id)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO product")).
			WillReturnError(assert.AnError)

		_, err := storage.CreateAd(ctx, req)
		assert.Error(t, err)
	})
}

func TestAdStorage_UpdateAd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	req := &dto.UpdateAdRequest{
		ID:         10,
		UserID:     1,
		CategoryID: 2,
		Title:      "Title",
		Status:     "active",
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product")).
			WithArgs(req.CategoryID, req.Title, req.Description, req.Price, req.Status, req.Location, req.ID, req.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product")).
			WithArgs(req.CategoryID, req.Title, req.Description, req.Price, req.Status, req.Location, req.ID, req.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.UpdateAd(ctx, req)
		assert.ErrorIs(t, err, ErrAdNotFound)
	})
}

func TestAdStorage_DeleteAd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	adID := int64(10)
	userID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET deleted_at")).
			WithArgs(adID, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.DeleteAd(ctx, adID, userID)
		assert.NoError(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET deleted_at")).
			WithArgs(adID, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.DeleteAd(ctx, adID, userID)
		assert.ErrorIs(t, err, ErrAdNotFound)
	})
}

func TestAdStorage_CloseAd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	adID := int64(10)
	userID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status = 'archived'")).
			WithArgs(adID, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.CloseAd(ctx, adID, userID)
		assert.NoError(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product SET status = 'archived'")).
			WithArgs(adID, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.CloseAd(ctx, adID, userID)
		assert.ErrorIs(t, err, ErrAdNotFound)
	})
}

func TestAdStorage_GetAdsByUserID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock)
	ctx := context.Background()
	userID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "seller_id", "category_id", "title", "description", "price", "status", "created_at", "updated_at", "photos", "views_count", "favorites_count",
		}).AddRow(int64(10), userID, int64(3), "Title", "Desc", int64(100), "active", now, now, []string{"p1.jpg"}, int64(10), int64(5))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(userID).
			WillReturnRows(rows)

		ads, err := storage.GetAdsByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, ads, 1)
		assert.Equal(t, userID, ads[0].SellerID)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnError(assert.AnError)

		_, err := storage.GetAdsByUserID(ctx, userID)
		assert.Error(t, err)
	})
}
