package handlers

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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
)

// ──────────────────────── Mocks ────────────────────────

type mockAuthService struct {
	LoginFunc                   func(ctx context.Context, email, password string) (string, string, models.User, error)
	ValidateTokenAndGetUserFunc func(ctx context.Context, tokenString string) (models.User, error)
	RegisterNewUserFunc         func(ctx context.Context, email, password, name string) (int64, error)
	LogoutFunc                  func(ctx context.Context, jti string, exp time.Time, refreshToken string) error
	RefreshFunc                 func(ctx context.Context, token string) (string, string, error)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, string, models.User, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password)
	}
	return "", "", models.User{}, nil
}
func (m *mockAuthService) ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error) {
	if m.ValidateTokenAndGetUserFunc != nil {
		return m.ValidateTokenAndGetUserFunc(ctx, tokenString)
	}
	return models.User{}, nil
}
func (m *mockAuthService) RegisterNewUser(ctx context.Context, email, password, name string) (int64, error) {
	if m.RegisterNewUserFunc != nil {
		return m.RegisterNewUserFunc(ctx, email, password, name)
	}
	return 0, nil
}
func (m *mockAuthService) Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, jti, exp, refreshToken)
	}
	return nil
}

func (m *mockAuthService) Refresh(ctx context.Context, token string) (string, string, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, token)
	}
	return "", "", nil
}

// ──────────────────────── Helpers ──────────────────────

func newTestAuthHandlers(authSvc Auth) *AuthHandlers {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	services := Services{Auth: authSvc}
	return NewAuthHandlers(log, services, time.Hour, time.Hour, "test-secret")
}

func testUser() models.User {
	return models.User{ID: 42, Email: "test@example.com", Name: "Test User"}
}

// ─────────────────── HandleLogin tests ─────────────────

// TestHandleLogin_WithValidCookie — успешный вход по cookie с токеном
func TestHandleLogin_WithValidCookie(t *testing.T) {
	user := testUser()
	token, err := jwt.NewToken(user, time.Hour, "test-secret")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	authSvc := &mockAuthService{
		ValidateTokenAndGetUserFunc: func(_ context.Context, _ string) (models.User, error) {
			return user, nil
		},
	}
	h := newTestAuthHandlers(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UserID != user.ID {
		t.Errorf("expected user_id %d, got %d", user.ID, body.UserID)
	}
}

// TestHandleLogin_WithInvalidCookie — вход по инвалидному токену → 401
func TestHandleLogin_WithInvalidCookie(t *testing.T) {
	authSvc := &mockAuthService{
		ValidateTokenAndGetUserFunc: func(_ context.Context, _ string) (models.User, error) {
			return models.User{}, errors.New("invalid token")
		},
	}
	h := newTestAuthHandlers(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "bad-token"})
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

// TestHandleLogin_Success — вход по email+password
func TestHandleLogin_Success(t *testing.T) {
	user := testUser()
	authSvc := &mockAuthService{
		LoginFunc: func(_ context.Context, _, _ string) (string, string, models.User, error) {
			return "some-jwt-token", "some-refresh-token", user, nil
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(LoginRequest{Email: "test@example.com", Password: "password1"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// TestHandleLogin_InvalidBody — невалидный JSON → 400
func TestHandleLogin_InvalidBody(t *testing.T) {
	h := newTestAuthHandlers(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{invalid-json"))
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

// TestHandleLogin_MissingFields — пустые email и password → 401 с validation errors
func TestHandleLogin_MissingFields(t *testing.T) {
	h := newTestAuthHandlers(&mockAuthService{})

	body, _ := json.Marshal(LoginRequest{Email: "", Password: ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}

	var ve ValidationErrors
	_ = json.NewDecoder(w.Result().Body).Decode(&ve)
	if ve.Email == "" || ve.Password == "" {
		t.Errorf("expected validation errors for email and password")
	}
}

// TestHandleLogin_InvalidCredentials — неверный пароль → 401
func TestHandleLogin_InvalidCredentials(t *testing.T) {
	authSvc := &mockAuthService{
		LoginFunc: func(_ context.Context, _, _ string) (string, string, models.User, error) {
			return "", "", models.User{}, auth.ErrInvalidCredentials
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(LoginRequest{Email: "test@example.com", Password: "password1"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

// TestHandleLogin_InternalError — ошибка сервиса → 500
func TestHandleLogin_InternalError(t *testing.T) {
	authSvc := &mockAuthService{
		LoginFunc: func(_ context.Context, _, _ string) (string, string, models.User, error) {
			return "", "", models.User{}, errors.New("db down")
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(LoginRequest{Email: "test@example.com", Password: "password1"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

// ─────────────────── HandleRegister tests ──────────────

// TestHandleRegister_Success — успешная регистрация и автологин
func TestHandleRegister_Success(t *testing.T) {
	user := testUser()
	authSvc := &mockAuthService{
		RegisterNewUserFunc: func(_ context.Context, _, _, _ string) (int64, error) {
			return user.ID, nil
		},
		LoginFunc: func(_ context.Context, _, _ string) (string, string, models.User, error) {
			return "some-jwt-token", "some-refresh-token", user, nil
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(RegisterRequest{Email: "test@example.com", Password: "password1", Name: "Test User"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// TestHandleRegister_InvalidBody — невалидный JSON → 400
func TestHandleRegister_InvalidBody(t *testing.T) {
	h := newTestAuthHandlers(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("{not-valid"))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

// TestHandleRegister_ValidationError — невалидный email → 400 с описанием ошибок
func TestHandleRegister_ValidationError(t *testing.T) {
	h := newTestAuthHandlers(&mockAuthService{})

	body, _ := json.Marshal(RegisterRequest{Email: "bad-email", Password: "short", Name: ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}

	var ve ValidationErrors
	_ = json.NewDecoder(w.Result().Body).Decode(&ve)
	if ve.Email == "" {
		t.Errorf("expected email validation error")
	}
}

// TestHandleRegister_UserAlreadyExists — попытка повторной регистрации → 400
func TestHandleRegister_UserAlreadyExists(t *testing.T) {
	authSvc := &mockAuthService{
		RegisterNewUserFunc: func(_ context.Context, _, _, _ string) (int64, error) {
			return 0, auth.ErrUserAlreadyExists
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(RegisterRequest{Email: "test@example.com", Password: "password1", Name: "Test User"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

// TestHandleRegister_InternalError — ошибка сервиса при регистрации → 500
func TestHandleRegister_InternalError(t *testing.T) {
	authSvc := &mockAuthService{
		RegisterNewUserFunc: func(_ context.Context, _, _, _ string) (int64, error) {
			return 0, errors.New("unexpected db error")
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(RegisterRequest{Email: "test@example.com", Password: "password1", Name: "Test User"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

// TestHandleRegister_AutoLoginFailed — успешная регистрация, но автологин упал → 500
func TestHandleRegister_AutoLoginFailed(t *testing.T) {
	authSvc := &mockAuthService{
		RegisterNewUserFunc: func(_ context.Context, _, _, _ string) (int64, error) {
			return 1, nil
		},
		LoginFunc: func(_ context.Context, _, _ string) (string, string, models.User, error) {
			return "", "", models.User{}, errors.New("autologin failed")
		},
	}
	h := newTestAuthHandlers(authSvc)

	body, _ := json.Marshal(RegisterRequest{Email: "test@example.com", Password: "password1", Name: "Test User"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleRegister(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

// ─────────────────── HandleLogout tests ────────────────

// TestHandleLogout_Success — успешный выход с jti в контексте
func TestHandleLogout_Success(t *testing.T) {
	called := false
	authSvc := &mockAuthService{
		LogoutFunc: func(_ context.Context, jti string, _ time.Time, _ string) error {
			called = true
			return nil
		},
	}
	h := newTestAuthHandlers(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(req.Context(), middleware.JtiKey, "test-jti-value")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
	if !called {
		t.Error("expected Logout service to be called")
	}
}

// TestHandleLogout_NoJtiInContext — нет jti в контексте → 500
func TestHandleLogout_NoJtiInContext(t *testing.T) {
	h := newTestAuthHandlers(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

// TestHandleLogout_ServiceError — ошибка сервиса при выходе → 500
func TestHandleLogout_ServiceError(t *testing.T) {
	authSvc := &mockAuthService{
		LogoutFunc: func(_ context.Context, _ string, _ time.Time, _ string) error {
			return errors.New("revoke failed")
		},
	}
	h := newTestAuthHandlers(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(req.Context(), middleware.JtiKey, "test-jti-value")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}
