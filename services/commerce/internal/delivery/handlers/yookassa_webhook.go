package handlers

//go:generate easyjson -all $GOFILE

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	paymentuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
)

const (
	opHandleYooKassaWebhook = "handlers.HandleYooKassaWebhook"
	webhookMaxBody          = 1 << 20 // 1 MiB
)

// PaymentApplier — usecase, способный применить терминальный статус платежа.
// Реализуется wallet.Usecase.
type PaymentApplier interface {
	ApplyProviderUpdate(ctx context.Context, provider, providerRef, status string, rawPayload []byte) error
}

// ProviderClient нужен webhook'у, чтобы re-fetch'ить статус платежа из ЮКассы
// (нельзя доверять payload'у — в нём может быть устаревший status).
type ProviderClient interface {
	GetPayment(ctx context.Context, providerRef string) (paymentuc.InitResult, error)
}

// YooKassaWebhookHandler принимает уведомления ЮКассы о смене статуса платежа.
type YooKassaWebhookHandler struct {
	log      *slog.Logger
	wallet   PaymentApplier
	provider ProviderClient
}

// NewYooKassaWebhookHandler создаёт обработчик webhook'а ЮКассы.
func NewYooKassaWebhookHandler(log *slog.Logger, wallet PaymentApplier, provider ProviderClient) *YooKassaWebhookHandler {
	return &YooKassaWebhookHandler{log: log, wallet: wallet, provider: provider}
}

// yookassaNotificationObject — payment-объект внутри уведомления.
type yookassaNotificationObject struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// yookassaNotification — минимально-необходимое представление webhook payload'а.
type yookassaNotification struct {
	Type   string                     `json:"type"`  // "notification"
	Event  string                     `json:"event"` // "payment.succeeded" / "payment.canceled" / ...
	Object yookassaNotificationObject `json:"object"`
}

// HandleWebhook парсит уведомление, re-fetch'ит статус у провайдера и применяет.
// Всегда возвращает 200, иначе ЮКасса будет ретраить (что нам не нужно — мы
// идемпотентны и сами доберём через reconciler).
func (h *YooKassaWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, webhookMaxBody))
	if err != nil {
		h.log.ErrorContext(r.Context(), "webhook read body failed",
			slog.String("op", opHandleYooKassaWebhook),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusOK) // не просим ретрай
		return
	}

	var note yookassaNotification
	if err := easyjson.Unmarshal(body, &note); err != nil || note.Object.ID == "" {
		h.log.WarnContext(r.Context(), "webhook payload invalid",
			slog.String("op", opHandleYooKassaWebhook),
			slog.String("error", errStr(err)),
		)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Re-fetch актуального статуса напрямую у ЮКассы.
	fresh, err := h.provider.GetPayment(r.Context(), note.Object.ID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "webhook get payment failed",
			slog.String("op", opHandleYooKassaWebhook),
			slog.String("provider_ref", note.Object.ID),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusOK) // reconciler доберёт
		return
	}

	if fresh.Status == models.PaymentStatusPending {
		// Ещё не финализирован — ничего не применяем.
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.wallet.ApplyProviderUpdate(
		r.Context(),
		models.PaymentProviderYooKassa,
		fresh.ProviderRef,
		fresh.Status,
		fresh.RawPayload,
	); err != nil {
		h.log.ErrorContext(r.Context(), "webhook apply failed",
			slog.String("op", opHandleYooKassaWebhook),
			slog.String("provider_ref", fresh.ProviderRef),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	var unmarshalErr *json.SyntaxError
	if errors.As(err, &unmarshalErr) {
		return unmarshalErr.Error()
	}
	return err.Error()
}
