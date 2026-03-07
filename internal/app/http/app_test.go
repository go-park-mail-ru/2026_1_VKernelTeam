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
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAds struct {
	mock.Mock
}

func (m *MockAds) GetAll() []models.Ad {
	args := m.Called()
	return args.Get(0).([]models.Ad)
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
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "test@test.com", "Password123").Return(int64(1), nil).Once()

		mockAuth.On("Login", mock.Anything, "test@test.com", "Password123").Return("fake-token-after-reg", nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=fake-token-after-reg")

		var resp RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := RegisterRequest{Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "invalid-email", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Short1"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "1234567890"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserExists", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "exist@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "exist@test.com", "Password123").Return(int64(0), storage.ErrUserExists).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "err@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "err@test.com", "Password123").Return(int64(0), errors.New("internal")).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuth.AssertExpectations(t)
	})
}

func TestHandleLogin(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "test@test.com", "Password123").Return("fake-token", nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=fake-token")
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := LoginRequest{Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := LoginRequest{Email: "not-an-email", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Short"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "PasswordNoNum"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "1234567890"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := LoginRequest{Email: "wrong@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "wrong@test.com", "Password123").Return("", auth.ErrInvalidCredentials).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := LoginRequest{Email: "err@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "err@test.com", "Password123").Return("", errors.New("internal")).Once()

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
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		rr := httptest.NewRecorder()

		mockAuth.On("Logout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=")
		mockAuth.AssertExpectations(t)
	})

	t.Run("NoCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/logout", nil)
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InvalidTokenSignature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/logout", nil)
		invalidToken, _ := ssntjwt.NewToken(user, time.Hour, "wrong-secret")
		req.AddCookie(&http.Cookie{Name: "token", Value: invalidToken})
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("AuthLogoutError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/logout", nil)
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

// Далее тесты для ручки получения объявлений

// тест успешного выполнения
func TestGetAdsHandler_Success(t *testing.T) {
	app, _, mockAds, _ := setupTestApp()

	testAds := []models.Ad{
		{ID: 1, Title: "Test Ad", Price: 100},
	}

	mockAds.On("GetAll").Return(testAds).Once()

	request, _ := http.NewRequest("GET", apiPrefix+"/ads", nil)
	rr := httptest.NewRecorder()

	app.router.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualData []models.Ad
	json.Unmarshal(rr.Body.Bytes(), &actualData)

	assert.Equal(t, testAds, actualData)
	mockAds.AssertExpectations(t)
}

// првоерка ограничения методов (обрабатываем только GET)
func TestGetAdsHandler_OnlyGet(t *testing.T) {
	// создаём чистые зависимости для теста
	repo := ads.NewAdsRepository()
	app := &App{
		services: Services{Ads: repo},
		log:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	// создаём POST запрос к эндпоинту
	request, err := http.NewRequest("POST", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку дял ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	app.handleGetAds(rr, request)

	// ожидаем код 405 - method not allowed
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST request, got %v", rr.Code)
	}
}

// првоерка, что сервер не падает при отсутствии объявлений
func TestGetAdsHandler_EmptyData(t *testing.T) {
	repo := &ads.AdsRepository{Data: []models.Ad{}}
	app := &App{
		services: Services{Ads: repo},
		log:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("GET", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку дял ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	app.handleGetAds(rr, request)

	// проверяем статус-код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 even with empty data, got %v", status)
	}

	// получаем тело ответа
	var actualData []models.Ad
	if err := json.Unmarshal(rr.Body.Bytes(), &actualData); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// првоеряем, что вернулся пустой массив, а не nil
	if len(actualData) != 0 {
		t.Errorf("expected 0 ads, got %d", len(actualData))
	}
}

// тестируем ошибку сервера
func TestGetAdsHandler_WrongMethod(t *testing.T) {
	// создаём чистые зависимости для теста
	repo := ads.NewAdsRepository()
	app := &App{
		services: Services{Ads: repo},
		log:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("POST", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку для ответа
	rr := httptest.NewRecorder()

	// вызываем handler
	app.handleGetAds(rr, request)

	// проверяем статус
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}
