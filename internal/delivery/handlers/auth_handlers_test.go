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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	ssntjwt "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleRegister(t *testing.T) {
	authH, _, mockAuth, _, _ := setupHandlers(t)

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password123", Name: "Test User"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: "test@test.com", Name: "Test User"}

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), "test@test.com", "Password123", "Test User").
			Return(int64(1), nil)

		mockAuth.EXPECT().
			Login(gomock.Any(), "test@test.com", "Password123").
			Return("fake-token-after-reg", user, nil)

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.Equal(t, "Test User", resp.Name)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer([]byte("{invalid}")))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		reqBody := RegisterRequest{Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "invalid-email", Password: "Password123", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Short1", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "1234567890", Name: "Test"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserExists", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "exist@test.com", Password: "Password123", Name: "Exist"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), "exist@test.com", "Password123", "Exist").
			Return(int64(0), auth.ErrUserAlreadyExists)

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "err@test.com", Password: "Password123", Name: "Err"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			RegisterNewUser(gomock.Any(), "err@test.com", "Password123", "Err").
			Return(int64(0), errors.New("internal"))

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("EmptyName", func(t *testing.T) {
		reqBody := RegisterRequest{Email: "test@test.com", Password: "Password123", Name: ""}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleLogin(t *testing.T) {
	authH, _, mockAuth, _, _ := setupHandlers(t)

	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		user := models.User{ID: 1, Email: "test@test.com", Name: "Test User"}

		mockAuth.EXPECT().
			Login(gomock.Any(), "test@test.com", "Password123").
			Return("fake-token", user, nil)

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.UserID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("TokenCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		// установить куку как будто пользователь уже вошёл
		req.AddCookie(&http.Cookie{Name: "token", Value: "existing-token"})
		rr := httptest.NewRecorder()

		user := models.User{ID: 2, Email: "cookie@test.com", Name: "Cookie User"}
		mockAuth.EXPECT().
			ValidateTokenAndGetUser(gomock.Any(), "existing-token").
			Return(user, nil)

		authH.HandleLogin(rr, req)

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
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "Short"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoDigit", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "PasswordNoNum"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("PasswordNoLetter", func(t *testing.T) {
		reqBody := LoginRequest{Email: "test@test.com", Password: "1234567890"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := LoginRequest{Email: "wrong@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Login(gomock.Any(), "wrong@test.com", "Password123").
			Return("", models.User{}, auth.ErrInvalidCredentials)

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		reqBody := LoginRequest{Email: "err@test.com", Password: "Password123"}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Login(gomock.Any(), "err@test.com", "Password123").
			Return("", models.User{}, errors.New("internal"))

		authH.HandleLogin(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleLogout(t *testing.T) {
	authH, _, mockAuth, _, _ := setupHandlers(t)

	user := models.User{ID: 1, Email: "test@test.com"}
	validToken, _ := ssntjwt.NewToken(user, time.Hour, "secret")

	t.Run("ValidRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		ctx := context.WithValue(req.Context(), middleware.JtiKey, "some-test-jti")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Logout(gomock.Any(), "some-test-jti", gomock.Any()).
			Return(nil)

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "token=")
	})

	t.Run("NoCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rr := httptest.NewRecorder()

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InvalidTokenSignature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		invalidToken, _ := ssntjwt.NewToken(user, time.Hour, "wrong-secret")
		req.AddCookie(&http.Cookie{Name: "token", Value: invalidToken})
		rr := httptest.NewRecorder()

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("AuthLogoutError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		ctx := context.WithValue(req.Context(), middleware.JtiKey, "test-jti-error")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			Logout(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(time.Time{})).
			Return(errors.New("err"))

		authH.HandleLogout(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
