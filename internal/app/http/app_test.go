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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	blacklist "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	ssntjwt "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAds struct {
	mock.Mock
}

func (m *MockAds) GetAllAds(ctx context.Context) ([]models.Ad, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Ad), args.Error(1)
}

type MockAuth struct {
	mock.Mock
}

func (m *MockAuth) Login(ctx context.Context, email string, password string) (string, models.User, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Get(1).(models.User), args.Error(2)
}

func (m *MockAuth) ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error) {
	args := m.Called(ctx, tokenString)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockAuth) RegisterNewUser(ctx context.Context, email string, password string, name string) (int64, error) {
	args := m.Called(ctx, email, password, name)
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

	// Собираем структуру handlers.Services, которую ожидает httpapp.New
	services := handlers.Services{
		Auth: mockAuth,
		Ads:  mockAds,
	}

	// Передаем структуру services вместо одного mockAuth
	app := New(logger, services, bl, 0, time.Hour, "secret")

	return app, mockAuth, mockAds, bl
}

func TestHandleRegister(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password123", Name: "Test User"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: "test@test.com", Name: "Test User"}

		mockAuth.On("RegisterNewUser", mock.Anything, "test@test.com", "Password123", "Test User").Return(int64(1), nil).Once()

		mockAuth.On("Login", mock.Anything, "test@test.com", "Password123").Return("fake-token-after-reg", user, nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.Equal(t, "Test User", resp.Name)
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
		reqBody := RegisterRequest{Email: "invalid-email", Password: "Password123", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Short1", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "1234567890", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserExists", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "exist@test.com", Password: "Password123", Name: "Exist"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "exist@test.com", "Password123", "Exist").Return(int64(0), auth.ErrUserAlreadyExists).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "err@test.com", Password: "Password123", Name: "Err"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("RegisterNewUser", mock.Anything, "err@test.com", "Password123", "Err").Return(int64(0), errors.New("internal")).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("EmptyName", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password123", Name: ""}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleLogin(t *testing.T) {
	app, mockAuth, _, _ := setupTestApp()

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: "test@test.com", Name: "Test User"}

		mockAuth.On("Login", mock.Anything, "test@test.com", "Password123").Return("fake-token", user, nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.UserID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("TokenCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", nil)
		// установить куку как будто пользователь уже вошёл
		req.AddCookie(&http.Cookie{Name: "token", Value: "existing-token"})
		rr := httptest.NewRecorder()

		user := models.User{ID: 2, Email: "cookie@test.com", Name: "Cookie User"}
		mockAuth.On("ValidateTokenAndGetUser", mock.Anything, "existing-token").Return(user, nil).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.UserID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := LoginRequest{Email: "not-an-email", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Short"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "PasswordNoNum"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "1234567890"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := LoginRequest{Email: "wrong@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "wrong@test.com", "Password123").Return("", models.User{}, auth.ErrInvalidCredentials).Once()

		app.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := LoginRequest{Email: "err@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.On("Login", mock.Anything, "err@test.com", "Password123").Return("", models.User{}, errors.New("internal")).Once()

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

	mockAds.On("GetAll", mock.Anything).Return(testAds, nil).Once()

	request, _ := http.NewRequest("GET", apiPrefix+"/ads", nil)
	rr := httptest.NewRecorder()

	app.router.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualData []models.Ad
	json.Unmarshal(rr.Body.Bytes(), &actualData)

	assert.Equal(t, testAds, actualData)
	mockAds.AssertExpectations(t)
}

// prvоерка ограничения методов (обрабатываем только GET)
func TestGetAdsHandler_OnlyGet(t *testing.T) {
	app, _, mockAds, _ := setupTestApp()
	_ = mockAds // POST не дойдёт до вызова GetAll

	// создаём POST запрос к эндпоинту
	request, err := http.NewRequest("POST", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку для ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	app.adsHandlers.HandleGetAds(rr, request)

	// ожидаем код 400 - method not allowed
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for POST request, got %v", rr.Code)
	}
}

// проверка, что сервер не падает при отсутствии объявлений
func TestGetAdsHandler_EmptyData(t *testing.T) {
	app, _, mockAds, _ := setupTestApp()
	mockAds.On("GetAll", mock.Anything).Return([]models.Ad{}, nil).Once()

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("GET", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку для ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	app.adsHandlers.HandleGetAds(rr, request)

	// проверяем статус-код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 even with empty data, got %v", status)
	}

	// получаем тело ответа
	var actualData []models.Ad
	if err := json.Unmarshal(rr.Body.Bytes(), &actualData); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// проверяем, что вернулся пустой массив, а не nil
	if len(actualData) != 0 {
		t.Errorf("expected 0 ads, got %d", len(actualData))
	}
	mockAds.AssertExpectations(t)
}

// тестируем неверный метод
func TestGetAdsHandler_WrongMethod(t *testing.T) {
	app, _, mockAds, _ := setupTestApp()
	_ = mockAds // DELETE не дойдёт до вызова GetAll

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("DELETE", apiPrefix+"/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку для ответа
	rr := httptest.NewRecorder()

	// вызываем handler
	app.adsHandlers.HandleGetAds(rr, request)

	// проверяем статус
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
