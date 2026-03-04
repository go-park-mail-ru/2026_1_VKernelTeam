package httpapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	ssntjwt "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/lib/jwt"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAds struct {
	mock.Mock
}

type MockAuth struct {
	mock.Mock
}

func (m *MockAuth) Login(ctx context.Context, email string, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuth) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuth) Logout(ctx context.Context, jti string, exp time.Time) error {
	args := m.Called(ctx, jti, exp)
	return args.Error(0)
}

func setupTestApp() (*App, *MockAuth, *MockAds, *blacklist.InMemory) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем моки для обоих сервисов
	mockAuth := new(MockAuth)
	mockAds := new(MockAds)

	bl := blacklist.New(time.Minute)

	// Собираем структуру Services, которую ожидает httpapp.New
	services := Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	// Передаем структуру services вместо одного mockAuth
	app := New(logger, services, bl, 8080, time.Hour, "secret")

	return app, mockAuth, mockAds, bl
}

func TestHandleRegister(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "test@test.com", "pass").Return(int64(1), nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var resp RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := RegisterRequest{Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserExists", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "exist@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "exist@test.com", "pass").Return(int64(0), storage.ErrUserExists).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "err@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "err@test.com", "pass").Return(int64(0), errors.New("internal")).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuth.AssertExpectations(t)
	})
}

func TestHandleLogin(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "test@test.com", "pass").Return("fake-token", nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=fake-token")
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := LoginRequest{Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := LoginRequest{Email: "wrong@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "wrong@test.com", "pass").Return("", auth.ErrInvalidCredentials).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := LoginRequest{Email: "err@test.com", Password: "pass"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "err@test.com", "pass").Return("", errors.New("internal")).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuth.AssertExpectations(t)
	})
}

func TestHandleLogout(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	user := models.User{ID: 1, Email: "test@test.com"}
	validToken, _ := ssntjwt.NewToken(user, time.Hour, "secret")

	t.Run("ValidRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		rr := httptest.NewRecorder()

		mockAuth.On("Logout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=")
		mockAuth.AssertExpectations(t)
	})

	t.Run("NoCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidTokenSignature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		invalidToken, _ := ssntjwt.NewToken(user, time.Hour, "wrong-secret")
		req.AddCookie(&http.Cookie{Name: "token", Value: invalidToken})
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("AuthLogoutError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		rr := httptest.NewRecorder()

		mockAuth.On("Logout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(errors.New("err")).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuth.AssertExpectations(t)
	})
}

func TestAppServerEndpoints(t *testing.T) {
	app, _, _, _ := setupTestApp()

	// Ensure Stop and MustRun logic are partially covered
	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	err := app.Run()
	assert.NoError(t, err) // Run should return nil on normal Stop

	app2, _, _, _ := setupTestApp()
	assert.Panics(t, func() {
		app2.srv.Addr = "invalid:port"
		app2.MustRun()
	})
}
