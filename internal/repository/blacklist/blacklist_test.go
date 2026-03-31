package blacklist

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRepositiry_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCache(ctrl)
	repo := New(mockCache)

	t.Run("successfully_add_to_blacklist", func(t *testing.T) {
		jti := "test-token-id"
		exp := time.Now().Add(time.Hour)
		key := "blacklist:" + jti

		mockCache.EXPECT().
			Set(gomock.Any(), key, "1", gomock.Any()).
			Do(func(ctx context.Context, k, v string, ttl time.Duration) {
				assert.InDelta(t, time.Hour.Seconds(), ttl.Seconds(), 2.0)
			}).
			Return(nil).
			Times(1)

		repo.Add(jti, exp)
	})

	t.Run("do_not_add_if_expired", func(t *testing.T) {
		jti := "expired-jti"
		exp := time.Now().Add(-time.Hour)

		mockCache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		repo.Add(jti, exp)
	})
}

func TestRepository_Check(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCache(ctrl)
	repo := New(mockCache)

	t.Run("token_exists_in_blacklist", func(t *testing.T) {
		jti := "blocked-jti"
		key := "blacklist:" + jti

		mockCache.EXPECT().Exists(gomock.Any(), key).Return(true).Times(1)

		result := repo.Check(jti)
		assert.True(t, result)
	})

	t.Run("token_not_in_blacklist", func(t *testing.T) {
		jti := "clean-jti"
		key := "blacklist:" + jti

		mockCache.EXPECT().Exists(gomock.Any(), key).Return(false).Times(1)

		result := repo.Check(jti)
		assert.False(t, result)
	})
}
