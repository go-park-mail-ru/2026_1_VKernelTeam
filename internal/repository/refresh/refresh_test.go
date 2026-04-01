package refresh_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh/mocks"
)

func TestRepository_SaveRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCache(ctrl)
	repo := refresh.New(mockCache)
	ctx := context.Background()

	t.Run("success_save", func(t *testing.T) {
		token := "some-uuid-token"
		userID := int64(42)
		ttl := time.Hour
		key := "refresh:" + token
		val := "42"

		// Проверяем, что в кэш уходит правильный ключ и строковое значение ID
		mockCache.EXPECT().Set(ctx, key, val, ttl).Return(nil)

		err := repo.SaveRefresh(ctx, token, userID, ttl)
		assert.NoError(t, err)
	})

	t.Run("cache_error", func(t *testing.T) {
		mockCache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("redis down"))

		err := repo.SaveRefresh(ctx, "token", int64(1), time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save refresh token")
	})
}

func TestRepository_GetRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCache(ctrl)
	repo := refresh.New(mockCache)
	ctx := context.Background()

	t.Run("success_get", func(t *testing.T) {
		token := "valid-token"
		key := "refresh:" + token

		// Имитируем, что в Redis лежит строка "100"
		mockCache.EXPECT().Get(ctx, key).Return("100", nil)

		userID, err := repo.GetRefresh(ctx, token)
		assert.NoError(t, err)
		assert.Equal(t, int64(100), userID)
	})

	t.Run("token_not_found", func(t *testing.T) {
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any()).
			Return("", redis.ErrNotFound)

		userID, err := repo.GetRefresh(ctx, "unknown")
		assert.Equal(t, int64(0), userID)
		assert.EqualError(t, err, "refresh token not found")
	})

	t.Run("invalid_data_format", func(t *testing.T) {
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any()).Return("not-a-number", nil)

		userID, err := repo.GetRefresh(ctx, "token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user id format")
		assert.Equal(t, int64(0), userID)
	})
}

func TestRepository_DeleteRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCache(ctrl)
	repo := refresh.New(mockCache)
	ctx := context.Background()

	t.Run("success_delete", func(t *testing.T) {
		token := "logout-token"
		key := "refresh:" + token

		mockCache.EXPECT().Delete(ctx, key).Return(nil)

		err := repo.DeleteRefresh(ctx, token)
		assert.NoError(t, err)
	})

	t.Run("delete_error", func(t *testing.T) {
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fmt.Errorf("del error"))

		err := repo.DeleteRefresh(ctx, "token")
		assert.Error(t, err)
	})
}
