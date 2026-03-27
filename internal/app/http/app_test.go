package httpapp

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type dummyTokenChecker struct{}

func (d dummyTokenChecker) Check(jti string) bool { return false }

func setupTestApp(t *testing.T) (*App, *mock_httpapp.MockAuth, *mock_httpapp.MockAds) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockAuth := mock_httpapp.NewMockAuth(ctrl)
	mockAds := mock_httpapp.NewMockAds(ctrl)

	services := Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	app := New(logger, services, dummyTokenChecker{}, 0, time.Hour, time.Hour, "secret")

	return app, mockAuth, mockAds
}

func TestAppServerEndpoints(t *testing.T) {
	app, _, _ := setupTestApp(t)

	// Тестируем нормальную остановку
	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	err := app.Run()
	assert.NoError(t, err)

	app2, _, _ := setupTestApp(t)
	assert.Panics(t, func() {
		app2.srv.Addr = ":-1" // Заведомо некорректный адрес для паники
		app2.MustRun()
	})
}
