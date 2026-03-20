package ads

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAll(t *testing.T) {
	testRepo := NewAdsRepository()

	results := testRepo.GetAll()

	assert.Len(t, results, 20)

	got := results[0]

	assert.Equal(t, int64(1), got.ID)
	assert.Equal(t, int64(101), got.SellerID)
	assert.Equal(t, int64(1), got.CategoryID)
	assert.Equal(t, "MacBook Pro 16 M1 Max", got.Title)
	assert.Equal(t, "Отличное состояние, полный комплект, использовался только для программирования. Батарея 95%.", got.Description)
	assert.Equal(t, int64(250000), got.Price)
	assert.Equal(t, "Москва", got.Location)
	assert.Equal(t, "active", got.Status)
	assert.NotZero(t, got.CreatedAt)
	assert.Equal(t, int64(150), got.ViewsCount)
	assert.Equal(t, int64(15), got.FavoritesCount)
}
