package handlers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/delivery/handlers/mocks"
	"github.com/golang/mock/gomock"
)

func setupHandlers(t *testing.T) (*AuthHandlers, *mocks.MockAuth) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAuth := mocks.NewMockAuth(ctrl)

	authH := NewAuthHandlers(logger, mockAuth, time.Hour, time.Hour, "test-secret")

	return authH, mockAuth
}
