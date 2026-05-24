package purchase

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

type fakeRepo struct {
	gotBuyer  int64
	gotCursor *int64
	gotLimit  int
	items     []dto.PurchaseItem
	err       error
}

func (f *fakeRepo) ListByBuyer(
	_ context.Context, buyerID int64, cursor *int64, limit int,
) ([]dto.PurchaseItem, error) {
	f.gotBuyer = buyerID
	f.gotCursor = cursor
	f.gotLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func newService(repo *fakeRepo) *Service {
	return NewService(repo, slog.Default())
}

func TestGetMyPurchases_DefaultsAndCursorProxy(t *testing.T) {
	repo := &fakeRepo{items: []dto.PurchaseItem{}}
	s := newService(repo)

	cur := int64(123)
	resp, err := s.GetMyPurchases(context.Background(), 7, &cur, 0)
	require.NoError(t, err)
	assert.Nil(t, resp.NextCursor)
	assert.Empty(t, resp.Purchases)

	assert.Equal(t, int64(7), repo.gotBuyer)
	assert.Equal(t, &cur, repo.gotCursor)
	// limit=0 → clamp to defaultListLim (20), look-ahead +1 = 21.
	assert.Equal(t, 21, repo.gotLimit)
}

func TestGetMyPurchases_ClampMaxLimit(t *testing.T) {
	repo := &fakeRepo{items: []dto.PurchaseItem{}}
	s := newService(repo)

	_, err := s.GetMyPurchases(context.Background(), 7, nil, 9999)
	require.NoError(t, err)
	// limit=9999 → clamp to maxListLim (50), +1 = 51.
	assert.Equal(t, 51, repo.gotLimit)
}

func TestGetMyPurchases_HasNextCursor(t *testing.T) {
	now := time.Now()
	repo := &fakeRepo{items: []dto.PurchaseItem{
		{OrderID: 5, PurchasedAt: now},
		{OrderID: 4, PurchasedAt: now},
		{OrderID: 3, PurchasedAt: now}, // sentinel for look-ahead
	}}
	s := newService(repo)

	resp, err := s.GetMyPurchases(context.Background(), 7, nil, 2)
	require.NoError(t, err)
	require.NotNil(t, resp.NextCursor)
	assert.Equal(t, int64(4), *resp.NextCursor)
	assert.Len(t, resp.Purchases, 2)
}

func TestGetMyPurchases_NoNextCursor(t *testing.T) {
	repo := &fakeRepo{items: []dto.PurchaseItem{{OrderID: 5}}}
	s := newService(repo)

	resp, err := s.GetMyPurchases(context.Background(), 7, nil, 2)
	require.NoError(t, err)
	assert.Nil(t, resp.NextCursor)
	assert.Len(t, resp.Purchases, 1)
}

func TestGetMyPurchases_StorageError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db down")}
	s := newService(repo)

	_, err := s.GetMyPurchases(context.Background(), 7, nil, 0)
	assert.Error(t, err)
}
