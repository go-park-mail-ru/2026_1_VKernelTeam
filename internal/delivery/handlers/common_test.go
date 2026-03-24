package handlers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/golang/mock/gomock"
)

// setupHandlers возвращает готовые хендлеры и моки без привязки к App
func setupHandlers(t *testing.T) (*AuthHandlers, *AdsHandlers, *mocks.MockAuth, *mocks.MockAds, *blacklist.InMemory) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAuth := mocks.NewMockAuth(ctrl)
	mockAds := mocks.NewMockAds(ctrl)
	bl := blacklist.New(time.Minute)

	services := Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	authH := NewAuthHandlers(logger, services, bl, time.Hour, "test-secret")
	adsH := NewAdsHandlers(logger, services)

	return authH, adsH, mockAuth, mockAds, bl
}
