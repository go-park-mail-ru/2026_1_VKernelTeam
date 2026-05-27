package ad

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

const (
	colID            = "id"
	colSellerID      = "seller_id"
	colCategoryID    = "category_id"
	colTitle         = "title"
	colDescription   = "description"
	colPrice         = "price"
	colStatus        = "status"
	colLocation      = "location"
	colLat           = "lat"
	colLon           = "lon"
	colCreatedAt     = "created_at"
	colUpdatedAt     = "updated_at"
	colPhotos        = "photos"
	colViewsCount    = "views_count"
	colFavorites     = "favorites_count"
	colIsBoosted     = "is_boosted"
	colIsHighlighted = "is_highlighted"
	colChangedAt     = "changed_at"
	photoP1          = "p1.jpg"
)

// expectEmptyCharacteristics добавляет mock-ожидания для двух запросов характеристик (пустые результаты).
func expectEmptyCharacteristics(mock pgxmock.PgxPoolIface, ids []int64) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"product_id", "name", "value"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"product_id", "name", "value"}))
}

func TestAdStorage_GetAdByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	adID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			colID, colSellerID, colCategoryID, colTitle, colDescription, colPrice, colStatus, colLocation, colLat, colLon, colCreatedAt, colUpdatedAt, colPhotos, colViewsCount, colFavorites, colIsBoosted, colIsHighlighted,
		}).AddRow(adID, int64(2), int64(3), "Title", "Desc", int64(100), "active", "Loc", nil, nil, now, now, []string{photoP1}, int64(10), int64(5), false, false)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(adID).
			WillReturnRows(rows)
		expectEmptyCharacteristics(mock, []int64{adID})

		ad, err := storage.GetAdByID(ctx, adID)
		assert.NoError(t, err)
		assert.Equal(t, adID, ad.ID)
		assert.Equal(t, []string{photoP1}, ad.Photos)
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

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			colID, colSellerID, colCategoryID, colTitle, colDescription, colPrice, colStatus, colLocation, colLat, colLon, colCreatedAt, colUpdatedAt, colPhotos, colViewsCount, colFavorites, colIsBoosted, colIsHighlighted,
		}).AddRow(int64(1), int64(2), int64(3), "Title", "Desc", int64(100), "active", "Loc", nil, nil, now, now, []string{photoP1}, int64(10), int64(5), false, false)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnRows(rows)
		expectEmptyCharacteristics(mock, []int64{1})

		ads, err := storage.GetAllAds(ctx)
		assert.NoError(t, err)
		assert.Len(t, ads, 1)
		assert.Equal(t, int64(1), ads[0].ID)
	})

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WillReturnRows(pgxmock.NewRows([]string{"id"}))
		// Empty result does not trigger characteristics loading (0 ads)

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

	storage := NewAdStorage(mock, slog.Default())
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
			WithArgs(req.UserID, req.CategoryID, req.Title, req.Description, req.Price, req.Status, req.Location, req.Lat, req.Lon).
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

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()

	t.Run("Success with only title", func(t *testing.T) {
		title := "New Title"
		req := &dto.UpdateAdRequest{
			ID:     10,
			UserID: 1,
			Title:  &title,
		}

		// Ожидаем динамический запрос с только title
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product")).
			WithArgs(title, req.ID, req.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Success with multiple fields", func(t *testing.T) {
		title := "New Title"
		price := int64(500)
		status := "active"
		req := &dto.UpdateAdRequest{
			ID:     10,
			UserID: 1,
			Title:  &title,
			Price:  &price,
			Status: &status,
		}

		// Ожидаем динамический запрос с несколькими полями
		// Порядок аргументов: title, price, status, id, userID
		mock.ExpectExec(regexp.QuoteMeta("UPDATE product")).
			WithArgs(title, price, status, req.ID, req.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		title := "Title"
		req := &dto.UpdateAdRequest{
			ID:     10,
			UserID: 1,
			Title:  &title,
		}

		mock.ExpectExec(regexp.QuoteMeta("UPDATE product")).
			WithArgs(title, req.ID, req.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.UpdateAd(ctx, req)
		assert.ErrorIs(t, err, ErrAdNotFound)
	})

	t.Run("Error when no fields to update", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     10,
			UserID: 1,
			// Все поля nil
		}

		err := storage.UpdateAd(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no fields to update")
	})
}

func TestAdStorage_DeleteAd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
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

	storage := NewAdStorage(mock, slog.Default())
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

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			colID, colSellerID, colCategoryID, colTitle, colDescription, colPrice, colStatus, colLocation, colLat, colLon, colCreatedAt, colUpdatedAt, colPhotos, colViewsCount, colFavorites, colIsBoosted, colIsHighlighted,
		}).AddRow(int64(10), userID, int64(3), "Title", "Desc", int64(100), "active", "Loc", nil, nil, now, now, []string{photoP1}, int64(10), int64(5), false, false)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(userID).
			WillReturnRows(rows)
		expectEmptyCharacteristics(mock, []int64{10})

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

func TestAdStorage_AddFavorite(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)
	adID := int64(10)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO favorite")).
			WithArgs(userID, adID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := storage.AddFavorite(ctx, userID, adID)
		assert.NoError(t, err)
	})

	t.Run("Conflict_Do_Nothing", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO favorite")).
			WithArgs(userID, adID).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := storage.AddFavorite(ctx, userID, adID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO favorite")).
			WithArgs(userID, adID).
			WillReturnError(assert.AnError)

		err := storage.AddFavorite(ctx, userID, adID)
		assert.Error(t, err)
	})
}

func TestAdStorage_RemoveFavorite(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)
	adID := int64(10)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM favorite")).
			WithArgs(userID, adID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := storage.RemoveFavorite(ctx, userID, adID)
		assert.NoError(t, err)
	})

	t.Run("NotFound_Still_Success", func(t *testing.T) {
		// Если записи не было, DELETE просто удалит 0 строк, это не ошибка
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM favorite")).
			WithArgs(userID, adID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := storage.RemoveFavorite(ctx, userID, adID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM favorite")).
			WithArgs(userID, adID).
			WillReturnError(assert.AnError)

		err := storage.RemoveFavorite(ctx, userID, adID)
		assert.Error(t, err)
	})
}

func TestAdStorage_GetUserFavorites(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		columns := []string{
			"id", "seller_id", "category_id", "title", "description", "price", "status",
			"location", "lat", "lon", "created_at", "updated_at", "photos", "views_count", "favorites_count",
			colIsBoosted, colIsHighlighted,
		}

		rows := pgxmock.NewRows(columns).
			AddRow(
				int64(101),
				int64(2),
				int64(3),
				"Title 1",
				"Desc 1",
				int64(100),
				"active",
				"Moscow",
				nil,
				nil,
				now,
				now,
				[]string{"img1.png"},
				int64(1),
				int64(1),
				false,
				false,
			).
			AddRow(
				int64(102),
				int64(2),
				int64(3),
				"Title 2",
				"Desc 2",
				int64(200),
				"active",
				"Piter",
				nil,
				nil,
				now,
				now,
				[]string{"img2.png"},
				int64(2),
				int64(2),
				false,
				false,
			)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(userID).
			WillReturnRows(rows)
		expectEmptyCharacteristics(mock, []int64{101, 102})

		ads, err := storage.GetUserFavorites(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, ads, 2)
		assert.Equal(t, int64(101), ads[0].ID)
		assert.Equal(t, "Moscow", ads[0].Location)
	})

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}))
		// Empty result does not trigger characteristics loading

		ads, err := storage.GetUserFavorites(ctx, userID)
		assert.NoError(t, err)
		assert.Empty(t, ads)
		assert.NotNil(t, ads)
	})

	t.Run("ScanError", func(t *testing.T) {
		// Подсовываем неверный тип данных (строку вместо id int64) для проверки обработки ошибки Scan
		rows := pgxmock.NewRows([]string{"id"}).AddRow("not_an_id")

		mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
			WithArgs(userID).
			WillReturnRows(rows)

		_, err := storage.GetUserFavorites(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan")
	})
}

func TestAdStorage_GetPriceHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	storage := NewAdStorage(mock, slog.Default())
	ctx := context.Background()
	adID := int64(1)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := pgxmock.NewRows([]string{colPrice, colChangedAt}).
			AddRow(int64(15000), now.Add(-48*time.Hour)).
			AddRow(int64(12000), now)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT price, changed_at")).
			WithArgs(adID).
			WillReturnRows(rows)

		history, err := storage.GetPriceHistory(ctx, adID)
		assert.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, int64(15000), history[0].Price)
		assert.Equal(t, int64(12000), history[1].Price)
	})

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT price, changed_at")).
			WithArgs(adID).
			WillReturnRows(pgxmock.NewRows([]string{colPrice, colChangedAt}))

		history, err := storage.GetPriceHistory(ctx, adID)
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT price, changed_at")).
			WithArgs(adID).
			WillReturnError(assert.AnError)

		_, err := storage.GetPriceHistory(ctx, adID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query")
	})

	t.Run("ScanError", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{colPrice, colChangedAt}).
			AddRow("not-a-number", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT price, changed_at")).
			WithArgs(adID).
			WillReturnRows(rows)

		_, err := storage.GetPriceHistory(ctx, adID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan")
	})
}
