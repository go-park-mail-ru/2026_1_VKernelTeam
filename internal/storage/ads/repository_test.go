package ads

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAll(t *testing.T) {
	testRepo := NewAdsRepository()

	results := testRepo.GetAll()

	assert.Len(t, results, 1)

	got := results[0]

	assert.Equal(t, int(1), got.ID)
	assert.Equal(t, "Продам гараж", got.Title)
	assert.Equal(t, "Очень ухоженный", got.Description)
	assert.Equal(t, 1_000_000, got.Price)
	assert.ElementsMatch(t, []string{"недвижимость", "гараж"}, got.Tags)
	assert.NotZero(t, got.CreatedAt)
	assert.Equal(t, int(1), got.SellerID)
	assert.Equal(t, int(12), got.Views)
}
