package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	paymentuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
)

type fakeApplier struct {
	calls       int
	gotProvider string
	gotRef      string
	gotStatus   string
	err         error
}

func (f *fakeApplier) ApplyProviderUpdate(_ context.Context, provider, ref, status string, _ []byte) error {
	f.calls++
	f.gotProvider, f.gotRef, f.gotStatus = provider, ref, status
	return f.err
}

type fakeProviderClient struct {
	res paymentuc.InitResult
	err error
}

func (f *fakeProviderClient) GetPayment(_ context.Context, _ string) (paymentuc.InitResult, error) {
	return f.res, f.err
}

func webhookLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestYooKassaWebhook_AppliesSucceeded(t *testing.T) {
	applier := &fakeApplier{}
	provider := &fakeProviderClient{res: paymentuc.InitResult{
		Status: models.PaymentStatusSucceeded, ProviderRef: "yoo-1", RawPayload: []byte("{}"),
	}}
	h := NewYooKassaWebhookHandler(webhookLogger(), applier, provider)

	body := `{"type":"notification","event":"payment.succeeded","object":{"id":"yoo-1","status":"succeeded"}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleWebhook(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, applier.calls)
	assert.Equal(t, models.PaymentProviderYooKassa, applier.gotProvider)
	assert.Equal(t, "yoo-1", applier.gotRef)
	assert.Equal(t, models.PaymentStatusSucceeded, applier.gotStatus)
}

// Если re-fetch вернул pending — ничего не применяем, отвечаем 200.
func TestYooKassaWebhook_SkipsPending(t *testing.T) {
	applier := &fakeApplier{}
	provider := &fakeProviderClient{res: paymentuc.InitResult{
		Status: models.PaymentStatusPending, ProviderRef: "yoo-2",
	}}
	h := NewYooKassaWebhookHandler(webhookLogger(), applier, provider)

	body := `{"type":"notification","event":"payment.waiting_for_capture","object":{"id":"yoo-2","status":"waiting_for_capture"}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleWebhook(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, applier.calls)
}

func TestYooKassaWebhook_ProviderError_StillReturns200(t *testing.T) {
	applier := &fakeApplier{}
	provider := &fakeProviderClient{err: errors.New("net down")}
	h := NewYooKassaWebhookHandler(webhookLogger(), applier, provider)

	body := `{"object":{"id":"yoo-3"}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleWebhook(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, applier.calls)
}

func TestYooKassaWebhook_InvalidPayload(t *testing.T) {
	applier := &fakeApplier{}
	provider := &fakeProviderClient{}
	h := NewYooKassaWebhookHandler(webhookLogger(), applier, provider)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", strings.NewReader(`{not-json`))
	rec := httptest.NewRecorder()
	h.HandleWebhook(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, applier.calls)
}
