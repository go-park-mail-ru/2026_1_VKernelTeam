package payment_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
)

func yooLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestYooKassa_InitPayment_Pending(t *testing.T) {
	var capturedAuth, capturedIdem, capturedCT, capturedBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payments", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		capturedAuth = r.Header.Get("Authorization")
		capturedIdem = r.Header.Get("Idempotence-Key")
		capturedCT = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		capturedBody = string(bodyBytes)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "yoo-uuid-1",
			"status": "pending",
			"amount": {"value": "500.00", "currency": "RUB"},
			"confirmation": {"type": "redirect", "confirmation_url": "https://yookassa/confirm/yoo-uuid-1"}
		}`))
	}))
	defer srv.Close()

	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "shop-1", SecretKey: "secret-1",
		APIURL: srv.URL, ReturnURL: "https://clover/return",
	}, srv.Client(), yooLogger())

	res, err := provider.InitPayment(context.Background(), models.Payment{
		ID: 42, UserID: 1, Amount: 500, Provider: models.PaymentProviderYooKassa,
	}, "idem-xyz")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusPending, res.Status)
	assert.Equal(t, "yoo-uuid-1", res.ProviderRef)
	assert.Equal(t, "https://yookassa/confirm/yoo-uuid-1", res.ConfirmationURL)
	assert.NotEmpty(t, res.RawPayload)

	// Заголовки.
	assert.True(t, strings.HasPrefix(capturedAuth, "Basic "), "Basic Auth expected")
	assert.Equal(t, "idem-xyz", capturedIdem)
	assert.Equal(t, "application/json", capturedCT)

	// Тело.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &parsed))
	assert.Equal(t, true, parsed["capture"])
	amount := parsed["amount"].(map[string]any)
	assert.Equal(t, "500.00", amount["value"])
	assert.Equal(t, "RUB", amount["currency"])
	conf := parsed["confirmation"].(map[string]any)
	assert.Equal(t, "redirect", conf["type"])
	assert.Equal(t, "https://clover/return", conf["return_url"])
	meta := parsed["metadata"].(map[string]any)
	assert.Equal(t, "1", meta["user_id"])
	assert.Equal(t, "42", meta["payment_id"])
}

func TestYooKassa_InitPayment_Succeeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"yoo-2","status":"succeeded","amount":{"value":"100.00","currency":"RUB"}}`))
	}))
	defer srv.Close()

	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "s", SecretKey: "k", APIURL: srv.URL, ReturnURL: "https://x",
	}, srv.Client(), yooLogger())

	res, err := provider.InitPayment(context.Background(), models.Payment{Amount: 100}, "i")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSucceeded, res.Status)
}

func TestYooKassa_InitPayment_Canceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"yoo-3","status":"canceled","amount":{"value":"50.00","currency":"RUB"}}`))
	}))
	defer srv.Close()

	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "s", SecretKey: "k", APIURL: srv.URL,
	}, srv.Client(), yooLogger())

	res, err := provider.InitPayment(context.Background(), models.Payment{Amount: 50}, "i")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusCancelled, res.Status)
}

func TestYooKassa_InitPayment_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "s", SecretKey: "k", APIURL: srv.URL,
	}, srv.Client(), yooLogger())

	_, err := provider.InitPayment(context.Background(), models.Payment{Amount: 100}, "i")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestYooKassa_GetPayment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payments/yoo-7", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"id":"yoo-7","status":"succeeded","amount":{"value":"100.00","currency":"RUB"}}`))
	}))
	defer srv.Close()

	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "s", SecretKey: "k", APIURL: srv.URL,
	}, srv.Client(), yooLogger())

	res, err := provider.GetPayment(context.Background(), "yoo-7")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSucceeded, res.Status)
	assert.Equal(t, "yoo-7", res.ProviderRef)
}

func TestYooKassa_GetPayment_EmptyRef(t *testing.T) {
	provider := payment.NewYooKassaProvider(payment.YooKassaConfig{
		ShopID: "s", SecretKey: "k", APIURL: "http://x",
	}, nil, yooLogger())
	_, err := provider.GetPayment(context.Background(), "")
	assert.Error(t, err)
}
