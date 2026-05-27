// Package payment — usecase платежей. Содержит контракт провайдера платежей,
// и реализации mock и yookassa.
package payment

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

// InitResult — результат инициализации платежа у провайдера.
//
//   - Status         — итоговый статус после InitPayment. Для синхронных провайдеров
//     (mock) это сразу 'succeeded'. Для асинхронных (ЮКасса) — 'pending', пока
//     пользователь не оплатит на стороне провайдера.
//   - ProviderRef    — внешний идентификатор платежа у провайдера. Используется
//     для повторных запросов GetPayment, идемпотентности webhook'ов и сверки.
//   - ConfirmationURL — URL, на который нужно редиректить пользователя для
//     завершения оплаты. Пустой для синхронных провайдеров.
//   - RawPayload     — сырой ответ провайдера (для аудита, может быть nil).
type InitResult struct {
	Status          string
	ProviderRef     string
	ConfirmationURL string
	RawPayload      []byte
}

// Provider — внешний платёжный сервис.
//
// Контракт асинхронный: InitPayment может вернуть Status='pending'. Финальный
// статус прилетает либо через webhook (см. delivery/handlers/yookassa_webhook),
// либо через периодический GetPayment в reconciler'е.
type Provider interface {
	// InitPayment создаёт платёж у провайдера. idempotencyKey передаётся как
	// идемпотентный заголовок (если провайдер поддерживает).
	InitPayment(ctx context.Context, p models.Payment, idempotencyKey string) (InitResult, error)

	// GetPayment запрашивает текущее состояние платежа у провайдера по providerRef.
	// Используется webhook-обработчиком (для re-fetch) и reconciler'ом.
	GetPayment(ctx context.Context, providerRef string) (InitResult, error)
}
