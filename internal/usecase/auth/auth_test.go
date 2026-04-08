// Тесты для пакета auth проверяют поведение сервиса аутентификации.
// Здесь определены моки для зависимостей и набор юнит-тестов на основные
// сценарии.
package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/user"
	mock_auth "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	storageMock.EXPECT().
		SaveUser(gomock.Any(), "existing@example.com", gomock.Any(), "Existing User").
		Return(int64(0), db.ErrUserExists).
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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	storageMock.EXPECT().
		User(gomock.Any(), email).
		Return(models.User{
			ID:       1,
			Email:    email,
			PassHash: passHash,
		}, nil).
		Times(1)

	refreshMock.EXPECT().
		SaveRefresh(gomock.Any(), gomock.Any(), int64(1), gomock.Any()).
		Return(nil).
		Times(1)

	token, _, _, err := auth.Login(context.Background(), email, password)

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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	storageMock.EXPECT().
		User(gomock.All(), email).
		Return(models.User{
			ID:       1,
			Email:    email,
			PassHash: passHash,
		}, nil).
		Times(1)

	_, _, _, err := auth.Login(context.Background(), email, "wrongpassword")

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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	storageMock.EXPECT().
		User(gomock.Any(), email).
		Return(models.User{}, db.ErrUserNotFound).
		Times(1)

	_, _, _, err := auth.Login(context.Background(), email, "password123")

	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
}

// TestRegisterNewUser_SaveError имитирует сбой при сохранении пользователя.
func TestRegisterNewUser_SaveError(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

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

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	storageMock.EXPECT().
		User(gomock.Any(), "user@example.com").
		Return(models.User{}, errors.New("something went wrong")).
		Times(1)

	_, _, _, err := auth.Login(context.Background(), "user@example.com", "pwd")
	if err == nil {
		t.Fatalf("expected error when user provider fails")
	}
}

// TestLogout_Success проверяет успешное добавление токена в список отозванных.
func TestLogout_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	jti := "my-jti"
	exp := time.Now().Add(time.Hour)

	tokenRevoker.EXPECT().
		Add(jti, exp).
		Times(1)

	err := auth.Logout(context.Background(), jti, exp, "")
	if err != nil {
		t.Fatalf("expected nil error on logout")
	}
}

func TestRefresh_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		storageMock,
		nil,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	oldRefresh := "old-uuid"
	userID := int64(42)

	// 1. Находим userID по токену
	refreshMock.EXPECT().
		GetRefresh(gomock.Any(), oldRefresh).
		Return(userID, nil)

	// 2. Удаляем старый
	refreshMock.EXPECT().
		DeleteRefresh(gomock.Any(), oldRefresh).
		Return(nil)

	// 3. Достаем юзера для генерации нового access
	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(models.User{ID: userID}, nil)

	// 4. Сохраняем новый refresh
	refreshMock.EXPECT().
		SaveRefresh(gomock.Any(), gomock.Any(), userID, gomock.Any()).
		Return(nil)

	access, refresh, err := auth.Refresh(context.Background(), oldRefresh)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

func TestRefresh_InvalidToken(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	auth := New(log,
		nil,
		nil,
		refreshMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	refreshMock.EXPECT().
		GetRefresh(gomock.Any(), "bad-token").
		Return(int64(0), errors.New("not found"))

	_, _, err := auth.Refresh(context.Background(), "bad-token")
	assert.Error(t, err)
}

func TestValidateTokenAndGetUser_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		time.Hour,
		time.Hour,
		testSecret,
	)

	user := models.User{ID: 10, Email: "user@test.com"}
	token, _ := jwt.NewToken(user, time.Hour, testSecret)

	storageMock.EXPECT().
		UserByID(gomock.Any(), int64(10)).
		Return(user, nil)

	result, err := auth.ValidateTokenAndGetUser(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
}

func TestValidateTokenAndGetUser_InvalidJWT(t *testing.T) {
	auth := New(
		getTestLogger(),
		nil,
		nil,
		nil,
		time.Hour,
		time.Hour,
		testSecret,
	)

	_, err := auth.ValidateTokenAndGetUser(context.Background(), "definitely-not-a-token")
	assert.Error(t, err)
}

func TestUpdateProfile_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		time.Hour,
		time.Hour,
		testSecret,
	)

	userID := int64(100)
	newName := "New Name"
	storageMock.EXPECT().
		UpdateUser(gomock.Any(), userID, newName).
		Return(models.User{ID: userID, Name: newName}, nil)

	u, err := auth.UpdateProfile(context.Background(), userID, newName)
	assert.NoError(t, err)
	assert.Equal(t, newName, u.Name)
}

func TestUpdateAvatar_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		time.Hour,
		time.Hour,
		testSecret,
	)

	userID := int64(124)
	filename := "test_avatar.png"

	// Имитируем валидный PNG (заголовок: 8 байт)
	pngHeader := []byte("\x89PNG\r\n\x1a\n")
	fileContent := append(pngHeader, []byte("fake-image-data")...)
	fileReader := bytes.NewReader(fileContent)

	// Ожидаем получение текущего пользователя для проверки старого аватара
	oldUser := models.User{
		ID:         userID,
		AvatarPath: "/static/img/avatars/old.png",
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(oldUser, nil).
		Times(1)

	// Ожидаем сохранение нового пути в БД
	storageMock.EXPECT().
		UpdateAvatarPath(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Ожидаем финальный возврат обновленного профиля
	updatedUser := models.User{
		ID:         userID,
		AvatarPath: "/static/img/avatars/124_123456789.png",
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(updatedUser, nil).
		Times(1)

	result, err := auth.UpdateAvatar(context.Background(), userID, fileReader, filename)

	assert.NoError(t, err)
	assert.Equal(t, updatedUser.AvatarPath, result.AvatarPath)

	defer os.RemoveAll("static")
}
