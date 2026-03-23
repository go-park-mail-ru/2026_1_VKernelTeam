package httpapp

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers" // Импортируем handlers
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers/mocks"
	blacklist "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupTestApp(t *testing.T) (*App, *mocks.MockAuth, *mocks.MockAds, *blacklist.InMemory) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAuth := mocks.NewMockAuth(ctrl)
	mockAds := mocks.NewMockAds(ctrl)

	bl := blacklist.New(time.Minute)

	services := handlers.Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	app := New(logger, services, bl, 0, time.Hour, "secret")

	return app, mockAuth, mockAds, bl
}

func TestAppServerEndpoints(t *testing.T) {
	app, _, _, _ := setupTestApp(t)

	// Тестируем нормальную остановку
	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	err := app.Run()
	assert.NoError(t, err)

	app2, _, _, _ := setupTestApp(t)
	assert.Panics(t, func() {
		app2.srv.Addr = ":-1" // Заведомо некорректный адрес для паники
		app2.MustRun()
	})
}
