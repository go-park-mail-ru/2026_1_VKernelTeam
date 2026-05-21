// Package payment — usecase платежей. Содержит интерфейс провайдера и mock-реализацию.
package payment

import (
	"context"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

// Provider — внешний платёжный сервис.
// MVP-реализация возвращает 'succeeded' мгновенно, без редиректа.
// Замена на ЮKassa/Stripe — без изменения вызывающего кода.
type Provider interface {
	// InitPayment получает черновик платежа и возвращает финальный статус и provider-ref.
	InitPayment(ctx context.Context, p models.Payment) (status string, providerRef string, err error)
}

// MockProvider — мгновенное успешное пополнение.
type MockProvider struct{}

// NewMockProvider создаёт мок-провайдера платежей, возвращающего мгновенный успех.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// InitPayment всегда возвращает succeeded и сгенерированный provider_ref.
func (m *MockProvider) InitPayment(_ context.Context, _ models.Payment) (string, string, error) {
	return models.PaymentStatusSucceeded, "mock-" + uuid.NewString(), nil
}
