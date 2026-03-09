// Тесты для пакета auth проверяют поведение сервиса аутентификации.
// Здесь определены моки для зависимостей и набор юнит-тестов на основные
// сценарии.
package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// Mock implementations for testing

type mockUserStorage struct {
	SaveUserFunc func(ctx context.Context, email string, passHash []byte) (int64, error)
	UserFunc     func(ctx context.Context, email string) (models.User, error)
	IsAdminFunc  func(ctx context.Context, userID int64) (bool, error)
}

func (m *mockUserStorage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	if m.SaveUserFunc != nil {
		return m.SaveUserFunc(ctx, email, passHash)
	}
	return 0, nil
}

func (m *mockUserStorage) User(ctx context.Context, email string) (models.User, error) {
	if m.UserFunc != nil {
		return m.UserFunc(ctx, email)
	}
	return models.User{}, nil
}

func (m *mockUserStorage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	if m.IsAdminFunc != nil {
		return m.IsAdminFunc(ctx, userID)
	}
	return false, nil
}

// Mock для TokenRevoker
type mockTokenRevoker struct {
	AddFunc func(jti string, exp time.Time)
}

func (m *mockTokenRevoker) Add(jti string, exp time.Time) {
	if m.AddFunc != nil {
		m.AddFunc(jti, exp)
	}
}

// Константа для тестов
const testSecret = "test-secret-key"

// getTestLogger возвращает простой логгер для использования в тестах.
func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// Tests

// TestRegisterNewUser_Success проверяет успешную регистрацию нового пользователя.
func TestRegisterNewUser_Success(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 1, nil
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	uid, err := auth.RegisterNewUser(context.Background(), "test@example.com", "password123")
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

	storageMock := &mockUserStorage{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 0, storage.ErrUserExists
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.RegisterNewUser(context.Background(), "existing@example.com", "password123")
	if err == nil {
		t.Fatalf("expected error for existing user")
	}

	if !errors.Is(err, storage.ErrUserExists) {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

// TestLogin_Success проверяет успешную аутентификацию и получение токена.
func TestLogin_Success(t *testing.T) {
	log := getTestLogger()
	password := "password123"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	storageMock := &mockUserStorage{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{
				ID:       1,
				Email:    email,
				PassHash: passHash,
			}, nil
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	token, err := auth.Login(context.Background(), "test@example.com", password)
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
	password := "password123"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	storageMock := &mockUserStorage{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{
				ID:       1,
				Email:    email,
				PassHash: passHash,
			}, nil
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.Login(context.Background(), "test@example.com", "wrongpassword")
	if err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
}

// TestLogin_UserNotFound проверяет, что запрос для несуществующего пользователя
// возвращает ошибку.
func TestLogin_UserNotFound(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{}, storage.ErrUserNotFound
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.Login(context.Background(), "nonexistent@example.com", "password123")
	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
}

// TestIsAdmin_True проверяет, что IsAdmin возвращает true для администратора.
func TestIsAdmin_True(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return true, nil
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("IsAdmin failed: %v", err)
	}

	if !isAdmin {
		t.Errorf("expected admin to be true")
	}
}

// TestIsAdmin_False проверяет, что IsAdmin возвращает false для обычного
// пользователя.
func TestIsAdmin_False(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, nil
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("IsAdmin failed: %v", err)
	}

	if isAdmin {
		t.Errorf("expected admin to be false")
	}
}

// TestIsAdmin_Error проверяет поведение при ошибке провайдера.
func TestIsAdmin_Error(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, storage.ErrUserNotFound
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.IsAdmin(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected error")
	}
}

// Additional tests for branching paths

// TestRegisterNewUser_SaveError имитирует сбой при сохранении пользователя.
func TestRegisterNewUser_SaveError(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 0, errors.New("db failure")
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.RegisterNewUser(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Fatalf("expected error when saving user fails")
	}
}

// TestLogin_UserProviderError проверяет реакцию на ошибку при получении данных
// пользователя.
func TestLogin_UserProviderError(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{}, errors.New("something went wrong")
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.Login(context.Background(), "user@example.com", "pwd")
	if err == nil {
		t.Fatalf("expected error when user provider fails")
	}
}

// TestIsAdmin_GenericError проверяет, что общая ошибка передаётся дальше.
func TestIsAdmin_GenericError(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, errors.New("whoops")
		},
	}

	tokenRevoker := &mockTokenRevoker{}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	_, err := auth.IsAdmin(context.Background(), 123)
	if err == nil {
		t.Fatalf("expected generic error from IsAdmin")
	}
}

func TestLogout_Success(t *testing.T) {
	log := getTestLogger()

	storageMock := &mockUserStorage{}

	tokenRevoker := &mockTokenRevoker{
		AddFunc: func(jti string, exp time.Time) {},
	}

	auth := New(log, storageMock, tokenRevoker, time.Hour, testSecret)

	err := auth.Logout(context.Background(), "my-jti", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("expected nil error on logout")
	}
}
