// Package auth реализует бизнес-логику аутентификации и авторизации
// пользователей. Он определяет сервис Auth с методами входа в систему,
// регистрации и проверки прав администратора, а также соответствующие
// интерфейсы для взаимодействия с хранилищем и провайдерами данных.
package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/google/uuid"

	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
	jwtlib "github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=auth.go -destination=mocks/mock_auth.go

const (
	opLogin                = "usecase.auth.Login"
	opValidateTokenAndUser = "usecase.auth.ValidateTokenAndGetUser"
	opRefresh              = "usecase.auth.Refresh"
	opLogout               = "usecase.auth.Logout"
	opRegisterNewUser      = "usecase.auth.RegisterNewUser"
	opGetProfile           = "usecase.auth.GetProfile"
	opUpdateProfile        = "usecase.auth.UpdateProfile"
	opUpdateAvatar         = "usecase.auth.UpdateAvatar"
)

var allowedTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

// TokenRevoker описывает интерфейс для отзыва токенов
type TokenRevoker interface {
	Add(jti string, exp time.Time)
}

// RefreshStorage интерфейс для работы с refresh токенами
type RefreshStorage interface {
	SaveRefresh(ctx context.Context, token string, userID int64, ttl time.Duration) error
	GetRefresh(ctx context.Context, token string) (int64, error)
	DeleteRefresh(ctx context.Context, token string) error
}

// FileStorage описывает интерфейс для загрузки файлов в объектное хранилище
type FileStorage interface {
	UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

// UserProviderSaver предоставляет методы сохранения пользователя в хранилище
// получения данных о пользователе и проверки его административных прав.
type UserProviderSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte, name string) (uid int64, err error)
	User(ctx context.Context, email string) (models.User, error)
	UserByID(ctx context.Context, userID int64) (models.User, error)
	UpdateUser(ctx context.Context, userID int64, name string) (models.User, error)
	UpdateAvatarPath(ctx context.Context, userID int64, path string) error
}

// Auth представляет собой сервис аутентификации. Он использует логгер,
// провайдеров пользователей и приложений, а также TTL для генерируемых
// токенов.
type Auth struct {
	log            *slog.Logger
	userStorage    UserProviderSaver
	tokenRevoker   TokenRevoker
	refreshStorage RefreshStorage
	fileStorage    FileStorage
	tokenTTL       time.Duration
	refreshTTL     time.Duration
	secret         string
}

// ErrInvalidCredentials возвращается, когда email/пароль не совпадают с
// сохранёнными данными.
// ErrUserAlreadyExists возвращается, когда пытаются создать пользователя с email, который уже существует.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// New создаёт новый экземпляр Auth с переданными зависимостями.
func New(
	log *slog.Logger,
	userStorage UserProviderSaver,
	tokenRevoker TokenRevoker,
	refreshStorage RefreshStorage,
	fileStorage FileStorage,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
	secret string,
) *Auth {
	return &Auth{
		log:            log,
		userStorage:    userStorage,
		tokenRevoker:   tokenRevoker,
		refreshStorage: refreshStorage,
		fileStorage:    fileStorage,
		tokenTTL:       tokenTTL,
		refreshTTL:     refreshTTL,
		secret:         secret,
	}
}

// Login аутентифицирует пользователя по email и паролю и возвращает JWT-токен.
func (a *Auth) Login(ctx context.Context, email, password string) (string, string, models.User, error) {
	a.log.InfoContext(ctx, "logging in user",
		slog.String("op", opLogin),
		slog.String("email", email),
	)

	user, err := a.userStorage.User(ctx, email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			a.log.WarnContext(ctx, "user not found",
				slog.String("op", opLogin),
				slog.String("email", email),
			)
			return "", "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
		}
		a.log.ErrorContext(ctx, "failed to get user",
			slog.String("op", opLogin),
			slog.String("error", err.Error()),
		)
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.WarnContext(ctx, "invalid credentials",
			slog.String("op", opLogin),
			slog.String("email", email),
		)
		return "", "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
	}

	token, err := jwt.NewToken(user, a.tokenTTL, a.secret)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to generate token",
			slog.String("op", opLogin),
			slog.String("error", err.Error()),
		)
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	refreshToken := uuid.New().String()
	err = a.refreshStorage.SaveRefresh(ctx, refreshToken, user.ID, a.refreshTTL)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to save refresh token",
			slog.String("op", opLogin),
			slog.String("error", err.Error()),
		)
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	a.log.InfoContext(ctx, "user logged in successfully",
		slog.String("op", opLogin),
		slog.Int64("user_id", user.ID),
	)
	return token, refreshToken, user, nil
}

// ValidateTokenAndGetUser проверяет валидность JWT-токена и возвращает данные пользователя
func (a *Auth) ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error) {
	a.log.DebugContext(ctx, "validating token",
		slog.String("op", opValidateTokenAndUser),
	)

	token, err := jwt.ParseToken(tokenString, a.secret)
	if err != nil {
		a.log.WarnContext(ctx, "failed to parse token",
			slog.String("op", opValidateTokenAndUser),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%w", err)
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		a.log.ErrorContext(ctx, "failed to extract claims from token",
			slog.String("op", opValidateTokenAndUser),
		)
		return models.User{}, errors.New("invalid token claims")
	}

	uidRaw, ok := claims["uid"].(float64)
	if !ok {
		a.log.ErrorContext(ctx, "invalid uid claim in token",
			slog.String("op", opValidateTokenAndUser),
		)
		return models.User{}, errors.New("invalid uid claim")
	}

	userID := int64(uidRaw)
	user, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			a.log.WarnContext(ctx, "user not found by token",
				slog.String("op", opValidateTokenAndUser),
				slog.Int64("user_id", userID),
			)
			return models.User{}, fmt.Errorf("user not found: %w", err)
		}
		a.log.ErrorContext(ctx, "failed to get user by id",
			slog.String("op", opValidateTokenAndUser),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%w", err)
	}

	a.log.InfoContext(ctx, "token validated successfully",
		slog.String("op", opValidateTokenAndUser),
		slog.Int64("user_id", userID),
	)
	return user, nil
}

// Refresh обновляет access-токен по refresh-токену
func (a *Auth) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	a.log.InfoContext(ctx, "refreshing tokens",
		slog.String("op", opRefresh),
	)

	userID, err := a.refreshStorage.GetRefresh(ctx, refreshToken)
	if err != nil {
		a.log.WarnContext(ctx, "invalid refresh token",
			slog.String("op", opRefresh),
			slog.String("error", err.Error()),
		)
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	_ = a.refreshStorage.DeleteRefresh(ctx, refreshToken)

	user, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get user by id",
			slog.String("op", opRefresh),
			slog.String("error", err.Error()),
		)
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	newAccess, err := jwt.NewToken(user, a.tokenTTL, a.secret)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to generate access token",
			slog.String("op", opRefresh),
			slog.String("error", err.Error()),
		)
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefresh := uuid.New().String()
	err = a.refreshStorage.SaveRefresh(ctx, newRefresh, userID, a.refreshTTL)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to save new refresh token",
			slog.String("op", opRefresh),
			slog.String("error", err.Error()),
		)
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	a.log.InfoContext(ctx, "tokens refreshed successfully",
		slog.String("op", opRefresh),
		slog.Int64("user_id", userID),
	)
	return newAccess, newRefresh, nil
}

func (a *Auth) Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error {
	a.log.InfoContext(ctx, "logging out user, revoking token",
		slog.String("op", opLogout),
		slog.String("jti", jti),
	)

	a.tokenRevoker.Add(jti, exp)

	if refreshToken != "" {
		_ = a.refreshStorage.DeleteRefresh(ctx, refreshToken)
	}

	a.log.InfoContext(ctx, "token successfully revoked and refresh deleted",
		slog.String("op", opLogout),
		slog.String("jti", jti),
	)
	return nil
}

// RegisterNewUser создаёт нового пользователя с указанным email и паролем.
func (a *Auth) RegisterNewUser(ctx context.Context, email, password, name string) (int64, error) {
	a.log.InfoContext(ctx, "registering user",
		slog.String("op", opRegisterNewUser),
		slog.String("email", email),
	)

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to generate password hash",
			slog.String("op", opRegisterNewUser),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("%s: %w", opRegisterNewUser, err)
	}

	id, err := a.userStorage.SaveUser(ctx, email, passHash, name)
	if err != nil {
		if errors.Is(err, db.ErrUserExists) {
			a.log.WarnContext(ctx, "user already exists",
				slog.String("op", opRegisterNewUser),
				slog.String("email", email),
			)
			return 0, fmt.Errorf("%s: %w", opRegisterNewUser, ErrUserAlreadyExists)
		}
		a.log.ErrorContext(ctx, "failed to save user",
			slog.String("op", opRegisterNewUser),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("%s: %w", opRegisterNewUser, err)
	}

	a.log.InfoContext(ctx, "user registered",
		slog.String("op", opRegisterNewUser),
		slog.Int64("user_id", id),
	)
	return id, nil
}

// GetProfile возвращает профиль пользователя по его ID.
func (a *Auth) GetProfile(ctx context.Context, userID int64) (models.User, error) {
	a.log.DebugContext(ctx, "getting user profile",
		slog.String("op", opGetProfile),
		slog.Int64("user_id", userID),
	)

	profile, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			a.log.WarnContext(ctx, "user not found",
				slog.String("op", opGetProfile),
				slog.Int64("user_id", userID),
			)
			return models.User{}, fmt.Errorf("%s: %w", opGetProfile, err)
		}
		a.log.ErrorContext(ctx, "failed to get user profile",
			slog.String("op", opGetProfile),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%s: %w", opGetProfile, err)
	}

	a.log.DebugContext(ctx, "user profile fetched",
		slog.String("op", opGetProfile),
		slog.Int64("user_id", userID),
	)
	return profile, nil
}

// UpdateProfile обновляет данные профиля пользователя.
func (a *Auth) UpdateProfile(ctx context.Context, userID int64, name string) (models.User, error) {
	a.log.InfoContext(ctx, "updating user profile",
		slog.String("op", opUpdateProfile),
		slog.Int64("user_id", userID),
	)

	user, err := a.userStorage.UpdateUser(ctx, userID, name)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			a.log.WarnContext(ctx, "user not found for update",
				slog.String("op", opUpdateProfile),
				slog.Int64("user_id", userID),
			)
			return models.User{}, fmt.Errorf("%s: %w", opUpdateProfile, err)
		}
		a.log.ErrorContext(ctx, "failed to update user profile",
			slog.String("op", opUpdateProfile),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%s: %w", opUpdateProfile, err)
	}

	a.log.InfoContext(ctx, "user profile updated successfully",
		slog.String("op", opUpdateProfile),
		slog.Int64("user_id", userID),
	)
	return user, nil
}

func (a *Auth) UpdateAvatar(ctx context.Context, userID int64, file multipart.File, filename string) (models.User, error) {
	a.log.InfoContext(ctx, "updating user avatar",
		slog.String("op", opUpdateAvatar),
		slog.Int64("user_id", userID),
	)

	currentUser, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return models.User{}, fmt.Errorf("%s: user not found: %w", opUpdateAvatar, err)
		}
		a.log.ErrorContext(ctx, "failed to fetch current user",
			slog.String("op", opUpdateAvatar),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%s: %w", opUpdateAvatar, err)
	}

	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		return models.User{}, fmt.Errorf("%s: failed to read file header: %w", opUpdateAvatar, err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return models.User{}, fmt.Errorf("%s: failed to seek file: %w", opUpdateAvatar, err)
	}

	contentType := http.DetectContentType(buff)
	if _, ok := allowedTypes[contentType]; !ok {
		a.log.WarnContext(ctx, "unsupported file type for avatar",
			slog.String("op", opUpdateAvatar),
			slog.String("content_type", contentType),
		)
		return models.User{}, fmt.Errorf("%s: unsupported file type: %s", opUpdateAvatar, contentType)
	}

	ext := filepath.Ext(filename)

	avatarURL, err := a.fileStorage.UploadFile(ctx, file, "avatars", ext)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to upload avatar to s3",
			slog.String("op", opUpdateAvatar),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%s: failed to upload avatar: %w", opUpdateAvatar, err)
	}

	if err := a.userStorage.UpdateAvatarPath(ctx, userID, avatarURL); err != nil {
		a.log.ErrorContext(ctx, "failed to update avatar path in db",
			slog.String("op", opUpdateAvatar),
			slog.String("error", err.Error()),
		)
		return models.User{}, fmt.Errorf("%s: failed to update avatar path: %w", opUpdateAvatar, err)
	}

	if currentUser.AvatarPath != "" {
		if err := a.fileStorage.DeleteFile(ctx, currentUser.AvatarPath); err != nil {
			a.log.WarnContext(ctx, "failed to delete old avatar from S3",
				slog.String("op", opUpdateAvatar),
				slog.String("old_avatar", currentUser.AvatarPath),
				slog.String("error", err.Error()),
			)
		}
	}

	a.log.InfoContext(ctx, "avatar updated successfully",
		slog.String("op", opUpdateAvatar),
		slog.Int64("user_id", userID),
	)
	return a.userStorage.UserByID(ctx, userID)
}
