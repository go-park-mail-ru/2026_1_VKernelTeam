package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/user"
	mock_auth "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/usecase/auth/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/pkg/jwt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret-key"

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestRegisterNewUser_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	tokenRevoker := mock_auth.NewMockTokenRevoker(ctrl)
	refreshMock := mock_auth.NewMockRefreshStorage(ctrl)

	email := "test@example.com"
	password := "password123"

	auth := New(log,
		storageMock,
		tokenRevoker,
		refreshMock,
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
		time.Hour,
		time.Hour,
		testSecret,
	)

	oldRefresh := "old-uuid"
	userID := int64(42)

	refreshMock.EXPECT().
		GetRefresh(gomock.Any(), oldRefresh).
		Return(userID, nil)

	refreshMock.EXPECT().
		DeleteRefresh(gomock.Any(), oldRefresh).
		Return(nil)

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(models.User{ID: userID}, nil)

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
		nil,
		nil,
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
	eventMock := mock_auth.NewMockEventPublisher(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		nil,
		eventMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	userID := int64(100)
	newName := "New Name"
	storageMock.EXPECT().
		UpdateUser(gomock.Any(), userID, newName).
		Return(models.User{ID: userID, Name: newName}, nil)
	eventMock.EXPECT().
		PublishUserUpdated(gomock.Any(), userID).
		Return(nil)

	u, err := auth.UpdateProfile(context.Background(), userID, newName)
	assert.NoError(t, err)
	assert.Equal(t, newName, u.Name)
}

func TestUpdateAvatar_Success(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	fileMock := mock_auth.NewMockFileStorage(ctrl)
	eventMock := mock_auth.NewMockEventPublisher(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		fileMock,
		eventMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	userID := int64(124)
	filename := "test_avatar.png"
	s3URL := "https://hb.vkcs.cloud/my-bucket/avatars/some-uuid.png"

	pngHeader := []byte("\x89PNG\r\n\x1a\n")
	fileContent := append(pngHeader, bytes.Repeat([]byte{0}, 504)...)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("avatar", filename)
	assert.NoError(t, err)
	_, err = part.Write(fileContent)
	assert.NoError(t, err)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	assert.NoError(t, err)
	defer form.RemoveAll()

	file, err := form.File["avatar"][0].Open()
	assert.NoError(t, err)
	defer file.Close()

	oldAvatarURL := "https://hb.vkcs.cloud/my-bucket/avatars/old-uuid.png"
	currentUser := models.User{
		ID:         userID,
		AvatarPath: oldAvatarURL,
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(currentUser, nil).
		Times(1)

	fileMock.EXPECT().
		UploadFile(gomock.Any(), gomock.Any(), "avatars", ".png").
		Return(s3URL, nil)

	storageMock.EXPECT().
		UpdateAvatarPath(gomock.Any(), userID, s3URL).
		Return(nil)

	fileMock.EXPECT().
		DeleteFile(gomock.Any(), oldAvatarURL).
		Return(nil)

	eventMock.EXPECT().
		PublishUserUpdated(gomock.Any(), userID).
		Return(nil)

	updatedUser := models.User{
		ID:         userID,
		AvatarPath: s3URL,
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(updatedUser, nil).
		Times(1)

	result, err := auth.UpdateAvatar(context.Background(), userID, file, filename)

	assert.NoError(t, err)
	assert.Equal(t, s3URL, result.AvatarPath)
}

func TestUpdateAvatar_DeleteOldFails(t *testing.T) {
	log := getTestLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mock_auth.NewMockUserProviderSaver(ctrl)
	fileMock := mock_auth.NewMockFileStorage(ctrl)
	eventMock := mock_auth.NewMockEventPublisher(ctrl)

	auth := New(log,
		storageMock,
		nil,
		nil,
		fileMock,
		eventMock,
		time.Hour,
		time.Hour,
		testSecret,
	)

	userID := int64(125)
	filename := "test_avatar.png"
	s3URL := "https://hb.vkcs.cloud/my-bucket/avatars/new-uuid.png"
	oldAvatarURL := "https://hb.vkcs.cloud/my-bucket/avatars/old-uuid.png"

	pngHeader := []byte("\x89PNG\r\n\x1a\n")
	fileContent := append(pngHeader, bytes.Repeat([]byte{0}, 504)...)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("avatar", filename)
	assert.NoError(t, err)
	_, err = part.Write(fileContent)
	assert.NoError(t, err)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	assert.NoError(t, err)
	defer form.RemoveAll()

	file, err := form.File["avatar"][0].Open()
	assert.NoError(t, err)
	defer file.Close()

	currentUser := models.User{
		ID:         userID,
		AvatarPath: oldAvatarURL,
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(currentUser, nil).
		Times(1)

	fileMock.EXPECT().
		UploadFile(gomock.Any(), gomock.Any(), "avatars", ".png").
		Return(s3URL, nil)

	storageMock.EXPECT().
		UpdateAvatarPath(gomock.Any(), userID, s3URL).
		Return(nil)

	fileMock.EXPECT().
		DeleteFile(gomock.Any(), oldAvatarURL).
		Return(errors.New("S3 deletion failed"))

	eventMock.EXPECT().
		PublishUserUpdated(gomock.Any(), userID).
		Return(nil)

	updatedUser := models.User{
		ID:         userID,
		AvatarPath: s3URL,
	}

	storageMock.EXPECT().
		UserByID(gomock.Any(), userID).
		Return(updatedUser, nil).
		Times(1)

	result, err := auth.UpdateAvatar(context.Background(), userID, file, filename)

	assert.NoError(t, err)
	assert.Equal(t, s3URL, result.AvatarPath)
}
