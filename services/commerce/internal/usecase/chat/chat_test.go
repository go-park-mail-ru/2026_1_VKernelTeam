package chat_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/chat"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/chat/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// активное объявление с заданными sellerID и price
func activeAd(id, sellerID, price int64) models.Ad {
	return models.Ad{
		ID:       id,
		SellerID: sellerID,
		Status:   models.AdStatusActive,
		Title:    "iPhone",
		Price:    price,
	}
}

func TestCreateOrderRequest(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 2, 1000), nil)
		chatMock.EXPECT().GetOrCreateChat(ctx, int64(38), int64(1), int64(2)).Return(int64(77), nil)
		chatMock.EXPECT().CreateMessage(ctx, gomock.Any()).Return(int64(555), nil)

		chatID, err := uc.CreateOrderRequest(ctx, 38, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(77), chatID)
	})

	t.Run("GetAdByID error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(models.Ad{}, errors.New("no ad"))

		_, err := uc.CreateOrderRequest(ctx, 38, 1)
		assert.Error(t, err)
	})

	t.Run("Ad is not active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		ad := activeAd(38, 2, 1000)
		ad.Status = models.AdStatusSold
		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(ad, nil)

		_, err := uc.CreateOrderRequest(ctx, 38, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("Buyer equals seller", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 1, 1000), nil)

		_, err := uc.CreateOrderRequest(ctx, 38, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "own product")
	})

	t.Run("GetOrCreateChat error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 2, 1000), nil)
		chatMock.EXPECT().GetOrCreateChat(ctx, int64(38), int64(1), int64(2)).
			Return(int64(0), errors.New("db err"))

		_, err := uc.CreateOrderRequest(ctx, 38, 1)
		assert.Error(t, err)
	})

	t.Run("CreateMessage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 2, 1000), nil)
		chatMock.EXPECT().GetOrCreateChat(ctx, int64(38), int64(1), int64(2)).Return(int64(77), nil)
		chatMock.EXPECT().CreateMessage(ctx, gomock.Any()).Return(int64(0), errors.New("db err"))

		_, err := uc.CreateOrderRequest(ctx, 38, 1)
		assert.Error(t, err)
	})
}

func TestConfirmPurchase(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatRow := models.Chat{ID: 5, AdID: 38, BuyerID: 1, SellerID: 2}
		chatMock.EXPECT().GetChatByID(ctx, int64(5)).Return(chatRow, nil)
		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 2, 1000), nil)
		chatMock.EXPECT().CompletePurchase(ctx, int64(5), int64(1), int64(38), int64(1000)).Return(nil)

		err := uc.ConfirmPurchase(ctx, 5, 2)
		assert.NoError(t, err)
	})

	t.Run("GetChatByID error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatMock.EXPECT().GetChatByID(ctx, int64(5)).Return(models.Chat{}, errors.New("no chat"))

		err := uc.ConfirmPurchase(ctx, 5, 2)
		assert.Error(t, err)
	})

	t.Run("User is not the seller", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatRow := models.Chat{ID: 5, AdID: 38, BuyerID: 1, SellerID: 2}
		chatMock.EXPECT().GetChatByID(ctx, int64(5)).Return(chatRow, nil)

		err := uc.ConfirmPurchase(ctx, 5, 1) // юзер 1 — покупатель, не продавец
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("Ad is not active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatRow := models.Chat{ID: 5, AdID: 38, BuyerID: 1, SellerID: 2}
		ad := activeAd(38, 2, 1000)
		ad.Status = models.AdStatusSold

		chatMock.EXPECT().GetChatByID(ctx, int64(5)).Return(chatRow, nil)
		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(ad, nil)

		err := uc.ConfirmPurchase(ctx, 5, 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("CompletePurchase error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatRow := models.Chat{ID: 5, AdID: 38, BuyerID: 1, SellerID: 2}
		chatMock.EXPECT().GetChatByID(ctx, int64(5)).Return(chatRow, nil)
		adMock.EXPECT().GetAdByID(ctx, int64(38)).Return(activeAd(38, 2, 1000), nil)
		chatMock.EXPECT().CompletePurchase(ctx, int64(5), int64(1), int64(38), int64(1000)).
			Return(errors.New("tx err"))

		err := uc.ConfirmPurchase(ctx, 5, 2)
		assert.Error(t, err)
	})
}

func TestGetAllChats(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		previews := []dto.ChatPreview{
			{ChatID: 1, Ad: dto.AdPreview{ID: 38}},
			{ChatID: 2, Ad: dto.AdPreview{ID: 39}},
		}
		chatMock.EXPECT().GetChatsByUserID(ctx, int64(1)).Return(previews, nil)

		resp, err := uc.GetAllChats(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, resp.Chats, 2)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatMock.EXPECT().GetChatsByUserID(ctx, int64(1)).Return(nil, errors.New("db err"))

		_, err := uc.GetAllChats(ctx, 1)
		assert.Error(t, err)
	})
}

func TestGetChat(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		detail := dto.ChatDetailResponse{
			ChatID:   5,
			Ad:       dto.AdPreview{ID: 38},
			Messages: []dto.MessageItem{{ID: 1}},
		}
		chatMock.EXPECT().GetChatDetail(ctx, int64(5), int64(1)).Return(detail, nil)

		resp, err := uc.GetChat(ctx, 5, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.ChatID)
	})

	t.Run("Error propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		chatMock := mocks.NewMockChatProvider(ctrl)
		adMock := mocks.NewMockAdProvider(ctrl)
		uc := chat.New(log, chatMock, adMock)

		chatMock.EXPECT().GetChatDetail(ctx, int64(5), int64(1)).
			Return(dto.ChatDetailResponse{}, errors.New("not found"))

		_, err := uc.GetChat(ctx, 5, 1)
		assert.Error(t, err)
	})
}
