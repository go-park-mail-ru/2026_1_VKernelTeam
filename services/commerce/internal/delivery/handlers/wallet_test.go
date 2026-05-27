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
	walletuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet"
)

func setupWalletHandlers(t *testing.T) (*WalletHandlers, *mocks.MockWalletService) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	m := mocks.NewMockWalletService(ctrl)
	return NewWalletHandlers(logger, m), m
}

func authedRequest(method, target string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequestWithContext(context.Background(), method, target, nil)
	} else {
		r = httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewBuffer(body))
	}
	if userID > 0 {
		ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
		r = r.WithContext(ctx)
	}
	return r
}

func TestHandleGetWallet_Success(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().GetBalance(gomock.Any(), int64(1)).
		Return(dto.WalletResponse{Balance: 500, Currency: "RUB"}, nil)

	rr := httptest.NewRecorder()
	h.HandleGetWallet(rr, authedRequest(http.MethodGet, "/api/v1/wallet", nil, 1))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.WalletResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, int64(500), resp.Balance)
	assert.Equal(t, "RUB", resp.Currency)
}

func TestHandleGetWallet_Unauthorized(t *testing.T) {
	h, _ := setupWalletHandlers(t)

	rr := httptest.NewRecorder()
	h.HandleGetWallet(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/wallet", nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleGetWallet_InternalError(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().GetBalance(gomock.Any(), int64(1)).
		Return(dto.WalletResponse{}, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleGetWallet(rr, authedRequest(http.MethodGet, "/api/v1/wallet", nil, 1))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleListWalletTransactions_Success(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().
		ListTransactions(gomock.Any(), int64(1), int64(50), 10).
		Return(dto.WalletTransactionListResponse{Items: []dto.WalletTransactionResponse{{ID: 1, Amount: 100}}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListWalletTransactions(rr, authedRequest(
		http.MethodGet, "/api/v1/wallet/transactions?limit=10&cursor=50", nil, 1,
	))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.WalletTransactionListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Items, 1)
	assert.Equal(t, int64(100), resp.Items[0].Amount)
}

func TestHandleListWalletTransactions_DefaultParams(t *testing.T) {
	h, m := setupWalletHandlers(t)
	// limit=0, cursor=0 — usecase сам нормализует.
	m.EXPECT().
		ListTransactions(gomock.Any(), int64(1), int64(0), 0).
		Return(dto.WalletTransactionListResponse{Items: []dto.WalletTransactionResponse{}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListWalletTransactions(rr, authedRequest(http.MethodGet, "/api/v1/wallet/transactions", nil, 1))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleListWalletTransactions_Unauthorized(t *testing.T) {
	h, _ := setupWalletHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleListWalletTransactions(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/wallet/transactions", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleTopupWallet_Success(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().Topup(gomock.Any(), int64(1), int64(500), "key-1").
		Return(dto.TopupWalletResponse{Balance: 500, PaymentID: 42}, nil)

	body, _ := json.Marshal(dto.TopupWalletRequest{Amount: 500, IdempotencyKey: "key-1"})
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, authedRequest(http.MethodPost, "/api/v1/wallet/topup", body, 1))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.TopupWalletResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, int64(500), resp.Balance)
	assert.Equal(t, int64(42), resp.PaymentID)
}

func TestHandleTopupWallet_Unauthorized(t *testing.T) {
	h, _ := setupWalletHandlers(t)
	body, _ := json.Marshal(dto.TopupWalletRequest{Amount: 100, IdempotencyKey: "k"})
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/wallet/topup", bytes.NewBuffer(body)))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleTopupWallet_BadJSON(t *testing.T) {
	h, _ := setupWalletHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, authedRequest(http.MethodPost, "/api/v1/wallet/topup", []byte("{not json"), 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errBody))
	assert.Equal(t, ErrInvalidRequestBody, errBody["error"])
}

func TestHandleTopupWallet_InvalidAmount(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().Topup(gomock.Any(), int64(1), int64(0), "k").
		Return(dto.TopupWalletResponse{}, walletuc.ErrInvalidAmount)

	body, _ := json.Marshal(dto.TopupWalletRequest{Amount: 0, IdempotencyKey: "k"})
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, authedRequest(http.MethodPost, "/api/v1/wallet/topup", body, 1))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errBody))
	assert.Equal(t, "INVALID_AMOUNT", errBody["error"])
}

func TestHandleTopupWallet_InvalidRequest(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().Topup(gomock.Any(), int64(1), int64(100), "").
		Return(dto.TopupWalletResponse{}, walletuc.ErrInvalidRequest)

	body, _ := json.Marshal(dto.TopupWalletRequest{Amount: 100, IdempotencyKey: ""})
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, authedRequest(http.MethodPost, "/api/v1/wallet/topup", body, 1))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errBody))
	assert.Equal(t, "INVALID_REQUEST", errBody["error"])
}

func TestHandleTopupWallet_InternalError(t *testing.T) {
	h, m := setupWalletHandlers(t)
	m.EXPECT().Topup(gomock.Any(), int64(1), int64(500), "k").
		Return(dto.TopupWalletResponse{}, errors.New("commit failed"))

	body, _ := json.Marshal(dto.TopupWalletRequest{Amount: 500, IdempotencyKey: "k"})
	rr := httptest.NewRecorder()
	h.HandleTopupWallet(rr, authedRequest(http.MethodPost, "/api/v1/wallet/topup", body, 1))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
