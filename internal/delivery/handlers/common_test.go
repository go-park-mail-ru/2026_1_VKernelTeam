package handlers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers/mocks"
	"github.com/golang/mock/gomock"
)

// setupHandlers возвращает готовые хендлеры и моки без привязки к App
func setupHandlers(t *testing.T) (*AuthHandlers, *AdsHandlers, *mocks.MockAuth, *mocks.MockAds) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAuth := mocks.NewMockAuth(ctrl)
	mockAds := mocks.NewMockAds(ctrl)

	services := Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	authH := NewAuthHandlers(logger, services, time.Hour, time.Hour, "test-secret")
	adsH := NewAdsHandlers(logger, services, time.Hour)

	return authH, adsH, mockAuth, mockAds
}

// setupCartHandlers возвращает готовый хендлер корзины и мок корзины
func setupCartHandlers(t *testing.T) (*CartHandlers, *mocks.MockCart) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCart := mocks.NewMockCart(ctrl)

	services := Services{
		Cart: mockCart,
	}

	cartH := NewCartHandlers(logger, services, time.Hour)

	return cartH, mockCart
}

// setupChatHandlers возвращает готовый хендлер чатов и мок сервиса Chat
func setupChatHandlers(t *testing.T) (*ChatHandlers, *mocks.MockChat) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockChat := mocks.NewMockChat(ctrl)

	services := &Services{
		Chat: mockChat,
	}

	chatHandlers := NewChatHandlers(logger, services)

	return chatHandlers, mockChat
}
