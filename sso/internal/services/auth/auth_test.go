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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// Mock implementations for testing

type mockUserSaver struct {
	SaveUserFunc func(ctx context.Context, email string, passHash []byte) (int64, error)
}

func (m *mockUserSaver) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	if m.SaveUserFunc != nil {
		return m.SaveUserFunc(ctx, email, passHash)
	}
	return 0, nil
}

type mockUserProvider struct {
	UserFunc    func(ctx context.Context, email string) (models.User, error)
	IsAdminFunc func(ctx context.Context, userID int64) (bool, error)
}

func (m *mockUserProvider) User(ctx context.Context, email string) (models.User, error) {
	if m.UserFunc != nil {
		return m.UserFunc(ctx, email)
	}
	return models.User{}, nil
}

func (m *mockUserProvider) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	if m.IsAdminFunc != nil {
		return m.IsAdminFunc(ctx, userID)
	}
	return false, nil
}

type mockAppProvider struct {
	AppFunc func(ctx context.Context, appID int64) (models.App, error)
}

func (m *mockAppProvider) App(ctx context.Context, appID int64) (models.App, error) {
	if m.AppFunc != nil {
		return m.AppFunc(ctx, appID)
	}
	return models.App{}, nil
}

// Helper function to create test logger
func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
// getTestLogger возвращает простой логгер для использования в тестах.

// Tests

func TestRegisterNewUser_Success(t *testing.T) {
	log := getTestLogger()
	userSaver := &mockUserSaver{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 1, nil
		},
	}
	userProvider := &mockUserProvider{}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	uid, err := auth.RegisterNewUser(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("RegisterNewUser failed: %v", err)
	}

	if uid != 1 {
		t.Errorf("expected uid 1, got %d", uid)
	}
}
// TestRegisterNewUser_Success проверяет успешную регистрацию нового пользователя.

func TestRegisterNewUser_UserExists(t *testing.T) {
	log := getTestLogger()
	userSaver := &mockUserSaver{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 0, storage.ErrUserExists
		},
	}
	userProvider := &mockUserProvider{}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.RegisterNewUser(context.Background(), "existing@example.com", "password123")
	if err == nil {
		t.Fatalf("expected error for existing user")
	}

	if !errors.Is(err, storage.ErrUserExists) {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}
// TestRegisterNewUser_UserExists проверяет, что при попытке зарегистрировать
// существующий email возвращается соответствующая ошибка.

func TestLogin_Success(t *testing.T) {
	log := getTestLogger()
	password := "password123"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{
				ID:       1,
				Email:    email,
				PassHash: passHash,
			}, nil
		},
	}
	appProvider := &mockAppProvider{
		AppFunc: func(ctx context.Context, appID int64) (models.App, error) {
			return models.App{
				ID:     appID,
				Name:   "test-app",
				Secret: "secret",
			}, nil
		},
	}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	token, err := auth.Login(context.Background(), "test@example.com", password, 1)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if token == "" {
		t.Fatalf("expected token, got empty string")
	}
}
// TestLogin_Success проверяет успешную аутентификацию и получение токена.

func TestLogin_InvalidCredentials(t *testing.T) {
	log := getTestLogger()
	password := "password123"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{
				ID:       1,
				Email:    email,
				PassHash: passHash,
			}, nil
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.Login(context.Background(), "test@example.com", "wrongpassword", 1)
	if err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
}
// TestLogin_InvalidCredentials убеждается, что неверный пароль приводит к ошибке.

func TestLogin_UserNotFound(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{}, storage.ErrUserNotFound
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.Login(context.Background(), "nonexistent@example.com", "password123", 1)
	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
}
// TestLogin_UserNotFound проверяет, что запрос для несуществующего пользователя
// возвращает ошибку.

func TestIsAdmin_True(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return true, nil
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("IsAdmin failed: %v", err)
	}

	if !isAdmin {
		t.Errorf("expected admin to be true")
	}
}
// TestIsAdmin_True проверяет, что IsAdmin возвращает true для администратора.

func TestIsAdmin_False(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, nil
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	isAdmin, err := auth.IsAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("IsAdmin failed: %v", err)
	}

	if isAdmin {
		t.Errorf("expected admin to be false")
	}
}
// TestIsAdmin_False проверяет, что IsAdmin возвращает false для обычного
// пользователя.

func TestIsAdmin_Error(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, storage.ErrAppNotFound
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.IsAdmin(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected error")
	}
}
// TestIsAdmin_Error проверяет поведение при ошибке провайдера.

// Additional tests for branching paths

func TestRegisterNewUser_SaveError(t *testing.T) {
	log := getTestLogger()
	userSaver := &mockUserSaver{
		SaveUserFunc: func(ctx context.Context, email string, passHash []byte) (int64, error) {
			return 0, errors.New("db failure")
		},
	}
	userProvider := &mockUserProvider{}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.RegisterNewUser(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Fatalf("expected error when saving user fails")
	}
}
// TestRegisterNewUser_SaveError имитирует сбой при сохранении пользователя.

func TestLogin_UserProviderError(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{}, errors.New("something went wrong")
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.Login(context.Background(), "user@example.com", "pwd", 1)
	if err == nil {
		t.Fatalf("expected error when user provider fails")
	}
}
// TestLogin_UserProviderError проверяет реакцию на ошибку при получении данных
// пользователя.

func TestLogin_AppProviderError(t *testing.T) {
	log := getTestLogger()

	password := "password123"
	passHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		UserFunc: func(ctx context.Context, email string) (models.User, error) {
			return models.User{ID: 1, Email: email, PassHash: passHash}, nil
		},
	}
	appProvider := &mockAppProvider{
		AppFunc: func(ctx context.Context, appID int64) (models.App, error) {
			return models.App{}, errors.New("app lookup failure")
		},
	}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.Login(context.Background(), "test@example.com", password, 42)
	if err == nil {
		t.Fatalf("expected error when app provider fails")
	}
}
// TestLogin_AppProviderError проверяет поведение при сбое провайдера приложения.

func TestIsAdmin_GenericError(t *testing.T) {
	log := getTestLogger()

	userSaver := &mockUserSaver{}
	userProvider := &mockUserProvider{
		IsAdminFunc: func(ctx context.Context, userID int64) (bool, error) {
			return false, errors.New("whoops")
		},
	}
	appProvider := &mockAppProvider{}

	auth := New(log, userSaver, userProvider, appProvider, time.Hour)

	_, err := auth.IsAdmin(context.Background(), 123)
	if err == nil {
		t.Fatalf("expected generic error from IsAdmin")
	}
}
// TestIsAdmin_GenericError проверяет, что общая ошибка передаётся дальше.
