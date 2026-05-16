package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	promouc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/promotion"
)

func setupPromotionHandlers(t *testing.T) (*PromotionHandlers, *mocks.MockPromotionService) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	m := mocks.NewMockPromotionService(ctrl)
	return NewPromotionHandlers(logger, m), m
}

// reqWithPath собирает запрос с подставленным {id} (через r.SetPathValue),
// что позволяет вызывать handler напрямую, минуя http.ServeMux.
func reqWithPath(method, target, idValue string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, bytes.NewBuffer(body))
	}
	r.SetPathValue("id", idValue)
	if userID > 0 {
		ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
		r = r.WithContext(ctx)
	}
	return r
}

// --- HandleGetPromotionPlans ---

func TestHandleGetPromotionPlans_Success(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().GetPlans(gomock.Any()).
		Return([]dto.PromotionPlanResponse{
			{ID: 1, Code: "boost_1d", Kind: "boost", DurationDays: 1, Price: 49},
		}, nil)

	rr := httptest.NewRecorder()
	h.HandleGetPromotionPlans(rr, httptest.NewRequest(http.MethodGet, "/api/v1/promotion/plans", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	var plans []dto.PromotionPlanResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&plans))
	require.Len(t, plans, 1)
	assert.Equal(t, "boost_1d", plans[0].Code)
}

func TestHandleGetPromotionPlans_InternalError(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().GetPlans(gomock.Any()).Return(nil, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleGetPromotionPlans(rr, httptest.NewRequest(http.MethodGet, "/api/v1/promotion/plans", nil))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- HandlePurchasePromotion ---

func TestHandlePurchasePromotion_Success(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().
		Purchase(gomock.Any(), int64(1), int64(10), "boost_7d", "uuid-1").
		Return(dto.PurchasePromotionResponse{
			Promotion: dto.PromotionResponse{
				ID: 555, ProductID: 10, Kind: models.PromotionKindBoost, PlanCode: "boost_7d",
			},
			WalletBalance: 801,
		}, nil)

	body, _ := json.Marshal(dto.PurchasePromotionRequest{
		PlanCode: "boost_7d", IdempotencyKey: "uuid-1",
	})
	rr := httptest.NewRecorder()
	h.HandlePurchasePromotion(rr, reqWithPath(http.MethodPost, "/api/v1/ads/10/promotions", "10", body, 1))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.PurchasePromotionResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, int64(555), resp.Promotion.ID)
	assert.Equal(t, int64(801), resp.WalletBalance)
}

func TestHandlePurchasePromotion_Unauthorized(t *testing.T) {
	h, _ := setupPromotionHandlers(t)
	body, _ := json.Marshal(dto.PurchasePromotionRequest{PlanCode: "x", IdempotencyKey: "k"})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/ads/10/promotions", bytes.NewBuffer(body))
	r.SetPathValue("id", "10")

	rr := httptest.NewRecorder()
	h.HandlePurchasePromotion(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandlePurchasePromotion_InvalidAdID(t *testing.T) {
	h, _ := setupPromotionHandlers(t)
	body, _ := json.Marshal(dto.PurchasePromotionRequest{PlanCode: "x", IdempotencyKey: "k"})

	rr := httptest.NewRecorder()
	h.HandlePurchasePromotion(rr, reqWithPath(http.MethodPost, "/api/v1/ads/abc/promotions", "abc", body, 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errBody))
	assert.Equal(t, ErrInvalidAdID, errBody["error"])
}

func TestHandlePurchasePromotion_BadJSON(t *testing.T) {
	h, _ := setupPromotionHandlers(t)
	rr := httptest.NewRecorder()
	h.HandlePurchasePromotion(rr, reqWithPath(http.MethodPost, "/api/v1/ads/10/promotions", "10", []byte("{bad"), 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// Тест маппинга всех бизнес-ошибок на корректные HTTP-коды и тела.
func TestHandlePurchasePromotion_ErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		ucErr      error
		wantStatus int
		wantCode   string
	}{
		{"AdNotFound", promouc.ErrAdNotFound, http.StatusNotFound, "AD_NOT_FOUND"},
		{"PlanNotFound", promouc.ErrPlanNotFound, http.StatusBadRequest, "PLAN_NOT_FOUND"},
		{"PlanInactive", promouc.ErrPlanInactive, http.StatusBadRequest, "PLAN_INACTIVE"},
		{"NotAdOwner", promouc.ErrNotAdOwner, http.StatusForbidden, "NOT_AD_OWNER"},
		{"InvalidAdStatus", promouc.ErrInvalidAdStatus, http.StatusBadRequest, "INVALID_AD_STATUS"},
		{"InsufficientFunds", promouc.ErrInsufficientFund, http.StatusBadRequest, "INSUFFICIENT_FUNDS"},
		{"InvalidRequest", promouc.ErrInvalidRequest, http.StatusBadRequest, "INVALID_REQUEST"},
		{"Unknown", errors.New("unexpected"), http.StatusInternalServerError, ErrInternalError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, m := setupPromotionHandlers(t)
			m.EXPECT().
				Purchase(gomock.Any(), int64(1), int64(10), "code", "k").
				Return(dto.PurchasePromotionResponse{}, tc.ucErr)

			body, _ := json.Marshal(dto.PurchasePromotionRequest{PlanCode: "code", IdempotencyKey: "k"})
			rr := httptest.NewRecorder()
			h.HandlePurchasePromotion(rr, reqWithPath(http.MethodPost, "/api/v1/ads/10/promotions", "10", body, 1))

			assert.Equal(t, tc.wantStatus, rr.Code)
			var errBody map[string]string
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&errBody))
			assert.Equal(t, tc.wantCode, errBody["error"])
		})
	}
}

// --- HandleListAdPromotions ---

func TestHandleListAdPromotions_Success(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().ListActiveByAd(gomock.Any(), int64(10)).
		Return([]dto.PromotionResponse{
			{ID: 1, ProductID: 10, Kind: models.PromotionKindBoost, PlanCode: "boost_7d"},
		}, nil)

	rr := httptest.NewRecorder()
	h.HandleListAdPromotions(rr, reqWithPath(http.MethodGet, "/api/v1/ads/10/promotions", "10", nil, 0))

	require.Equal(t, http.StatusOK, rr.Code)
	var items []dto.PromotionResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&items))
	require.Len(t, items, 1)
}

func TestHandleListAdPromotions_InvalidID(t *testing.T) {
	h, _ := setupPromotionHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleListAdPromotions(rr, reqWithPath(http.MethodGet, "/api/v1/ads/x/promotions", "x", nil, 0))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleListAdPromotions_InternalError(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().ListActiveByAd(gomock.Any(), int64(10)).
		Return(nil, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleListAdPromotions(rr, reqWithPath(http.MethodGet, "/api/v1/ads/10/promotions", "10", nil, 0))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- HandleListUserPromotions ---

func TestHandleListUserPromotions_Success(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().ListByUser(gomock.Any(), int64(1), int64(50), 5).
		Return(dto.PromotionListResponse{
			Items: []dto.PromotionResponse{{ID: 1, PlanCode: "boost_7d"}},
		}, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/profile/promotions?limit=5&cursor=50", nil)
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, int64(1))
	r = r.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.HandleListUserPromotions(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.PromotionListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Items, 1)
}

func TestHandleListUserPromotions_Unauthorized(t *testing.T) {
	h, _ := setupPromotionHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleListUserPromotions(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profile/promotions", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleListUserPromotions_InternalError(t *testing.T) {
	h, m := setupPromotionHandlers(t)
	m.EXPECT().ListByUser(gomock.Any(), int64(1), int64(0), 0).
		Return(dto.PromotionListResponse{}, errors.New("db down"))

	r := httptest.NewRequest(http.MethodGet, "/api/v1/profile/promotions", nil)
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, int64(1))
	r = r.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.HandleListUserPromotions(rr, r)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
