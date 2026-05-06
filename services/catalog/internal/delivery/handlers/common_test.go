package handlers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/handlers/mocks"
	"github.com/golang/mock/gomock"
)

// setupAdsHandlers создаёт *AdsHandlers с моками. Используется во всех ads-тестах.
func setupAdsHandlers(t *testing.T) (*AdsHandlers, *mocks.MockAds) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAds := mocks.NewMockAds(ctrl)
	adsH := NewAdsHandlers(logger, mockAds)
	return adsH, mockAds
}

// setupViewsHandlers создаёт *ViewsHandlers с моком Views.
func setupViewsHandlers(t *testing.T) (*ViewsHandlers, *mocks.MockViews) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockViews := mocks.NewMockViews(ctrl)
	viewsH := NewViewsHandlers(logger, mockViews)
	return viewsH, mockViews
}
