package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

type fakePurchaseProvider struct {
	gotBuyer  int64
	gotCursor *int64
	gotLimit  int
	resp      dto.PurchaseListResponse
	err       error
}

func (f *fakePurchaseProvider) GetMyPurchases(
	_ context.Context, buyerID int64, cursor *int64, limit int,
) (dto.PurchaseListResponse, error) {
	f.gotBuyer = buyerID
	f.gotCursor = cursor
	f.gotLimit = limit
	return f.resp, f.err
}

func newPurchaseHandlers() (*PurchaseHandlers, *fakePurchaseProvider) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	fake := &fakePurchaseProvider{}
	return NewPurchaseHandlers(logger, fake), fake
}

func reqMyPurchases(target string, userID int64) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if userID > 0 {
		ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
		r = r.WithContext(ctx)
	}
	return r
}

func TestHandleListMyPurchases_Unauthorized(t *testing.T) {
	h, _ := newPurchaseHandlers()

	rr := httptest.NewRecorder()
	h.HandleListMyPurchases(rr, reqMyPurchases("/api/v1/profile/purchases", 0))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleListMyPurchases_Happy(t *testing.T) {
	h, fake := newPurchaseHandlers()

	now := time.Now()
	chatID := int64(78)
	fake.resp = dto.PurchaseListResponse{
		Purchases: []dto.PurchaseItem{
			{
				OrderID: 101, ProductID: 38, Title: "iPhone", Price: 35000,
				Photo: "p.jpg", Location: "Москва",
				Seller:      dto.SellerPreview{ID: 42, Name: "Иван"},
				Source:      dto.PurchaseSourceChat,
				PurchasedAt: now,
				ChatID:      &chatID,
			},
		},
	}

	rr := httptest.NewRecorder()
	h.HandleListMyPurchases(rr, reqMyPurchases("/api/v1/profile/purchases?cursor=500&limit=10", 7))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.PurchaseListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Purchases, 1)
	assert.Equal(t, int64(101), resp.Purchases[0].OrderID)
	assert.Equal(t, "chat", resp.Purchases[0].Source)
	require.NotNil(t, resp.Purchases[0].ChatID)
	assert.Equal(t, int64(78), *resp.Purchases[0].ChatID)

	assert.Equal(t, int64(7), fake.gotBuyer)
	require.NotNil(t, fake.gotCursor)
	assert.Equal(t, int64(500), *fake.gotCursor)
	assert.Equal(t, 10, fake.gotLimit)
}

func TestHandleListMyPurchases_NoCursorNoLimit(t *testing.T) {
	h, fake := newPurchaseHandlers()

	rr := httptest.NewRecorder()
	h.HandleListMyPurchases(rr, reqMyPurchases("/api/v1/profile/purchases", 7))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, fake.gotCursor)
	assert.Equal(t, 0, fake.gotLimit)
}

func TestHandleListMyPurchases_InvalidCursorIgnored(t *testing.T) {
	h, fake := newPurchaseHandlers()

	rr := httptest.NewRecorder()
	h.HandleListMyPurchases(rr, reqMyPurchases("/api/v1/profile/purchases?cursor=abc", 7))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, fake.gotCursor)
}

func TestHandleListMyPurchases_UsecaseError(t *testing.T) {
	h, fake := newPurchaseHandlers()
	fake.err = errors.New("db down")

	rr := httptest.NewRecorder()
	h.HandleListMyPurchases(rr, reqMyPurchases("/api/v1/profile/purchases", 7))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
