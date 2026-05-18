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

	status, ref, err := provider.InitPayment(context.Background(), models.Payment{
		UserID:   1,
		Amount:   500,
		Provider: models.PaymentProviderMock,
	})

	assert.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSucceeded, status)
	assert.True(t, strings.HasPrefix(ref, "mock-"))
	assert.Greater(t, len(ref), len("mock-"))
}

func TestMockProvider_InitPayment_UniqueRefPerCall(t *testing.T) {
	provider := payment.NewMockProvider()

	_, ref1, _ := provider.InitPayment(context.Background(), models.Payment{Amount: 100})
	_, ref2, _ := provider.InitPayment(context.Background(), models.Payment{Amount: 100})

	assert.NotEqual(t, ref1, ref2)
}
