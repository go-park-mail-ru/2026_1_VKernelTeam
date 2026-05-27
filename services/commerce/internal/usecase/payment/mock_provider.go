package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

// MockProvider — синхронный успешный платёж. Используется в dev-окружении
// без боевых ключей ЮКассы.
type MockProvider struct{}

// NewMockProvider создаёт мок-провайдера платежей, возвращающего мгновенный успех.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// InitPayment всегда возвращает succeeded и сгенерированный provider_ref.
// ConfirmationURL пустой — фронту делать редирект не нужно.
func (m *MockProvider) InitPayment(_ context.Context, _ models.Payment, _ string) (InitResult, error) {
	return InitResult{
		Status:      models.PaymentStatusSucceeded,
		ProviderRef: "mock-" + uuid.NewString(),
	}, nil
}

// GetPayment у мок-провайдера не имеет смысла: все платежи сразу терминальны.
// Возвращаем ошибку, чтобы reconciler/webhook никогда не звали этот метод по mock-платежам.
func (m *MockProvider) GetPayment(_ context.Context, _ string) (InitResult, error) {
	return InitResult{}, errors.New("mock provider does not support GetPayment")
}
