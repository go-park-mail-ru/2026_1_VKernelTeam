package handlers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/handlers/mocks"
	"github.com/golang/mock/gomock"
)

const (
	keyProductID = "product_id"
	titleIPhone  = "iPhone"
	nameIvan     = "Иван"
)

func setupCartHandlers(t *testing.T) (*CartHandlers, *mocks.MockCart) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCart := mocks.NewMockCart(ctrl)
	return NewCartHandlers(logger, mockCart), mockCart
}

func setupChatHandlers(t *testing.T) (*ChatHandlers, *mocks.MockChat) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockChat := mocks.NewMockChat(ctrl)
	return NewChatHandlers(logger, mockChat), mockChat
}
