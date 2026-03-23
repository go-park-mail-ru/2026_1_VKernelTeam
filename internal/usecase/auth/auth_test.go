// Тесты для пакета auth проверяют поведение сервиса аутентификации.
// Здесь определены моки для зависимостей и набор юнит-тестов на основные
// сценарии.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	storage "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository"
	mock_auth "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth/mocks"
	"github.com/golang/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

// Константа для тестов
const testSecret = "test-secret-key"

// getTestLogger возвращает простой логгер для использования в тестах.
func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// TestRegisterNewUser_Success проверяет успешную регистрацию нового пользователя.
func TestRegisterNewUser_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	email := "test@example.com"
	password := "password123"

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		SaveUser(gomock.Any(), email, gomock.Any(), "Test User").
		Return(int64(1), nil).
		Times(1)

	uid, err := auth.RegisterNewUser(context.Background(), email, password, "Test User")

	if err != nil {
		t.Fatalf("RegisterNewUser failed: %v", err)
	}

	if uid != 1 {
		t.Errorf("expected uid 1, got %d", uid)
	}
}

// TestRegisterNewUser_UserExists проверяет, что при попытке зарегистрировать
// существующий email возвращается соответствующая ошибка.
func TestRegisterNewUser_UserExists(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		SaveUser(gomock.Any(), "existing@example.com", gomock.Any(), "Existing User").
		Return(int64(0), storage.ErrUserExists).
		Times(1)

	_, err := auth.RegisterNewUser(context.Background(), "existing@example.com", "password123", "Existing User")
	if err == nil {
		t.Fatalf("expected error for existing user")
	}

	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

// TestLogin_Success проверяет успешную аутентификацию и получение токена.
func TestLogin_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	password := "password123"
	email := "test@example.com"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		User(gomock.Any(), email).
		Return(models.User{
			ID:       1,
			Email:    email,
			PassHash: passHash,
		}, nil).
		Times(1)

	token, _, err := auth.Login(context.Background(), email, password)

	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if token == "" {
		t.Fatalf("expected token, got empty string")
	}
}

// TestLogin_InvalidCredentials убеждается, что неверный пароль приводит к ошибке.
func TestLogin_InvalidCredentials(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	password := "password123"
	email := "test@example.com"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		User(gomock.All(), email).
		Return(models.User{
			ID:       1,
			Email:    email,
			PassHash: passHash,
		}, nil).
		Times(1)

	_, _, err := auth.Login(context.Background(), email, "wrongpassword")

	if err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
}

// TestLogin_UserNotFound проверяет, что запрос для несуществующего пользователя
// возвращает ошибку.
func TestLogin_UserNotFound(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	email := "nonexistent@example.com"

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		User(gomock.Any(), email).
		Return(models.User{}, storage.ErrUserNotFound).
		Times(1)

	_, _, err := auth.Login(context.Background(), email, "password123")

	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
}

// TestIsAdmin_True проверяет, что IsAdmin возвращает true для администратора.
func TestIsAdmin_True(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		IsAdmin(gomock.Any(), int64(1)).
		Return(true, nil).
		Times(1)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)

	if err != nil || !isAdmin {
		t.Fatalf("IsAdmin failed: %v", err)
	}
}

// TestIsAdmin_False проверяет, что IsAdmin возвращает false для обычного
// пользователя.
func TestIsAdmin_False(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		IsAdmin(gomock.Any(), int64(1)).
		Return(false, nil).
		Times(1)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)

	if err != nil || isAdmin {
		t.Fatalf("IsAdmin failed: %v", err)
	}
}

// TestIsAdmin_Error проверяет поведение при ошибке провайдера.
func TestIsAdmin_Error(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		IsAdmin(gomock.Any(), gomock.Any()).
		Return(false, storage.ErrUserNotFound).
		Times(1)

	_, err := auth.IsAdmin(context.Background(), 1)
	if !errors.Is(err, storage.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// TestRegisterNewUser_SaveError имитирует сбой при сохранении пользователя.
func TestRegisterNewUser_SaveError(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		SaveUser(gomock.Any(), "test@example.com", gomock.Any(), "Test User").
		Return(int64(0), errors.New("db failure")).
		Times(1)

	_, err := auth.RegisterNewUser(context.Background(), "test@example.com", "password123", "Test User")
	if err == nil {
		t.Fatalf("expected error when saving user fails")
	}
}

// TestLogin_UserProviderError проверяет реакцию на ошибку при получении данных пользователя.
func TestLogin_UserProviderError(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		User(gomock.Any(), "user@example.com").
		Return(models.User{}, errors.New("something went wrong")).
		Times(1)

	_, _, err := auth.Login(context.Background(), "user@example.com", "pwd")
	if err == nil {
		t.Fatalf("expected error when user provider fails")
	}
}

// TestIsAdmin_GenericError проверяет, что общая ошибка передаётся дальше.
func TestIsAdmin_GenericError(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	storageMock.EXPECT().
		IsAdmin(gomock.Any(), int64(123)).
		Return(false, errors.New("whoops")).
		Times(1)

	_, err := auth.IsAdmin(context.Background(), 123)
	if err == nil {
		t.Fatalf("expected generic error from IsAdmin")
	}
}

// TestLogout_Success проверяет успешное добавление токена в список отозванных.
func TestLogout_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	jti := "my-jti"
	exp := time.Now().Add(time.Hour)

	tokenRevoker.EXPECT().
		Add(jti, exp).
		Times(1)

	err := auth.Logout(context.Background(), jti, exp)
	if err != nil {
		t.Fatalf("expected nil error on logout")
	}
}
