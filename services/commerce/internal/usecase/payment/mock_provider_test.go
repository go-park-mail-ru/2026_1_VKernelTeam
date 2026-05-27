package payment_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
)

func TestMockProvider_InitPayment(t *testing.T) {
	provider := payment.NewMockProvider()

	res, err := provider.InitPayment(context.Background(), models.Payment{
		UserID:   1,
		Amount:   500,
		Provider: models.PaymentProviderMock,
	}, "idem-1")

	assert.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSucceeded, res.Status)
	assert.True(t, strings.HasPrefix(res.ProviderRef, "mock-"))
	assert.Greater(t, len(res.ProviderRef), len("mock-"))
	assert.Empty(t, res.ConfirmationURL)
}

func TestMockProvider_InitPayment_UniqueRefPerCall(t *testing.T) {
	provider := payment.NewMockProvider()

	r1, _ := provider.InitPayment(context.Background(), models.Payment{Amount: 100}, "a")
	r2, _ := provider.InitPayment(context.Background(), models.Payment{Amount: 100}, "b")

	assert.NotEqual(t, r1.ProviderRef, r2.ProviderRef)
}

func TestMockProvider_GetPayment_Unsupported(t *testing.T) {
	provider := payment.NewMockProvider()
	_, err := provider.GetPayment(context.Background(), "ref")
	assert.Error(t, err)
}
