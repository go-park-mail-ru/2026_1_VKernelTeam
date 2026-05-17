package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
	ssntjwt "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/jwt"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/usecase/auth"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testPassword = "Password123"
	testName     = "Test"
	testUserName = "Test User"
	testEmail    = "test@test.com"
)

func TestHandleRegister(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Password: testPassword, Name: testUserName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: testEmail, Name: testUserName}

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), testEmail, testPassword, testUserName).
			Return(int64(1), nil)

		mockAuth.EXPECT().
			Login(gomock.Any(), testEmail, testPassword).
			Return("fake-token-after-reg", "fake-refresh", user, nil)

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		cookies := rr.Result().Cookies()
		var hasToken, hasCsrf bool
		for _, c := range cookies {
			if c.Name == cookieNameToken {
				hasToken = true
				assert.Equal(t, "fake-token-after-reg", c.Value)
			}
			if c.Name == cookieNameCSRF {
				hasCsrf = true
				assert.NotEmpty(t, c.Value)
			}
		}
		assert.True(t, hasToken, "кука 'token' не найдена")
		assert.True(t, hasCsrf, "кука 'csrf_token' не найдена")

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.Equal(t, testUserName, resp.Name)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Password: testPassword}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: "invalid-email", Password: testPassword, Name: testName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Name: testName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Password: "Short1", Name: testName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Password: "Password", Name: testName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Password: "1234567890", Name: testName}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserExists", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: "exist@test.com", Password: testPassword, Name: "Exist"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), "exist@test.com", testPassword, "Exist").
			Return(int64(0), auth.ErrUserAlreadyExists)

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: "err@test.com", Password: testPassword, Name: "Err"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), "err@test.com", testPassword, "Err").
			Return(int64(0), errors.New("internal"))

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("EmptyName", func(t *testing.T) {
		reqBody := dto.RegisterRequest{Email: testEmail, Password: testPassword, Name: ""}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleLogin(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: testEmail, Password: testPassword}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: testEmail, Name: testUserName}

		mockAuth.EXPECT().
			Login(gomock.Any(), testEmail, testPassword).
			Return("fake-token", "fake-refresh", user, nil)

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		cookies := rr.Result().Cookies()
		var hasToken, hasCsrf bool
		for _, c := range cookies {
			if c.Name == cookieNameToken {
				hasToken = true
				assert.Equal(t, "fake-token", c.Value)
			}
			if c.Name == cookieNameCSRF {
				hasCsrf = true
				assert.NotEmpty(t, c.Value)
			}
		}
		assert.True(t, hasToken, "кука 'token' не найдена")
		assert.True(t, hasCsrf, "кука 'csrf_token' не найдена")

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.UserID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("TokenCookie", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: "existing-token"})
		rr := httptest.NewRecorder()

		user := models.User{ID: 2, Email: "cookie@test.com", Name: "Cookie User"}
		mockAuth.EXPECT().
			ValidateTokenAndGetUser(gomock.Any(), "existing-token").
			Return(user, nil)

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.UserID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: "not-an-email", Password: testPassword}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: testEmail}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: testEmail, Password: "Short"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: testEmail, Password: "PasswordNoNum"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: testEmail, Password: "1234567890"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: "wrong@test.com", Password: testPassword}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Login(gomock.Any(), "wrong@test.com", testPassword).
			Return("", "", models.User{}, auth.ErrInvalidCredentials)

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := dto.LoginRequest{Email: "err@test.com", Password: testPassword}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Login(gomock.Any(), "err@test.com", testPassword).
			Return("", "", models.User{}, errors.New("internal"))

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleLogout(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	user := models.User{ID: 1, Email: testEmail}
	validToken, _ := ssntjwt.NewToken(user, time.Hour, "secret")

	const testCsrf = "test-csrf-token-123"

	t.Run("ValidRequest", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: validToken})

		req.AddCookie(&http.Cookie{Name: cookieNameCSRF, Value: testCsrf})
		req.Header.Set("X-CSRF-Token", testCsrf)

		ctx := context.WithValue(req.Context(), middleware.JtiKey, "some-test-jti")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Logout(gomock.Any(), "some-test-jti", gomock.Any(), gomock.Any()).
			Return(nil)

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=")
	})

	t.Run("NoCookie", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", nil)
		rr := httptest.NewRecorder()

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("AuthLogoutError", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: validToken})

		req.AddCookie(&http.Cookie{Name: cookieNameCSRF, Value: testCsrf})
		req.Header.Set("X-CSRF-Token", testCsrf)

		ctx := context.WithValue(req.Context(), middleware.JtiKey, "test-jti-error")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Logout(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(time.Time{}), gomock.Any()).
			Return(errors.New("err")).AnyTimes()

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
