// Package auth реализует бизнес-логику аутентификации и авторизации
// пользователей. Он определяет сервис Auth с методами входа в систему,
// регистрации и проверки прав администратора, а также соответствующие
// интерфейсы для взаимодействия с хранилищем и провайдерами данных.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/google/uuid"

	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/database"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/jwt"
	jwtlib "github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

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

// UserProviderSaver предоставляет методы сохранения пользователя в хранилище
// получения данных о пользователе и проверки его административных прав.
type UserProviderSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte, name string) (uid int64, err error)
	User(ctx context.Context, email string) (models.User, error)
	UserByID(ctx context.Context, userID int64) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// Auth представляет собой сервис аутентификации. Он использует логгер,
// провайдеров пользователей и приложений, а также TTL для генерируемых
// токенов.
type Auth struct {
	log            *slog.Logger
	userStorage    UserProviderSaver
	tokenRevoker   TokenRevoker
	refreshStorage RefreshStorage
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
	tokenTTL time.Duration,
	refreshTTL time.Duration,
	secret string,
) *Auth {
	return &Auth{
		log:            log,
		userStorage:    userStorage,
		tokenRevoker:   tokenRevoker,
		refreshStorage: refreshStorage,
		tokenTTL:       tokenTTL,
		refreshTTL:     refreshTTL,
		secret:         secret,
	}
}

// Login аутентифицирует пользователя по email и паролю и возвращает JWT-токен. В случае
// ошибок возвращается описанная ошибка.
func (a *Auth) Login(ctx context.Context, email, password string) (string, string, models.User, error) {

	log := a.log.With(slog.String("email", email))
	log.Info("logging in user")

	user, err := a.userStorage.User(ctx, email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			log.Error("user not found")
			return "", "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
		}
		log.Error("failed to get user")
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials")
		return "", "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
	}

	token, err := jwt.NewToken(user, a.tokenTTL, a.secret)
	if err != nil {
		log.Error("failed to generate token")
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	refreshToken := uuid.New().String()
	err = a.refreshStorage.SaveRefresh(ctx, refreshToken, user.ID, a.refreshTTL)
	if err != nil {
		log.Error("failed to save refresh token")
		return "", "", models.User{}, fmt.Errorf("%w", err)
	}

	log.Info("user logged in")
	return token, refreshToken, user, nil
}

// ValidateTokenAndGetUser проверяет валидность JWT-токена и возвращает данные пользователя
func (a *Auth) ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error) {
	const op = "auth.ValidateTokenAndGetUser"
	log := a.log.With(slog.String("op", op))

	token, err := jwt.ParseToken(tokenString, a.secret)
	if err != nil {
		log.Info("failed to parse token", slog.String("error", err.Error()))
		return models.User{}, fmt.Errorf("%w", err)
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		log.Error("failed to extract claims from token")
		return models.User{}, errors.New("invalid token claims")
	}

	uidRaw, ok := claims["uid"].(float64)
	if !ok {
		log.Error("invalid uid claim in token")
		return models.User{}, errors.New("invalid uid claim")
	}

	userID := int64(uidRaw)
	user, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			log.Info("user not found", slog.Int64("user_id", userID))
			return models.User{}, fmt.Errorf("user not found: %w", err)
		}
		log.Error("failed to get user by id", slog.String("error", err.Error()))
		return models.User{}, fmt.Errorf("%w", err)
	}

	log.Info("token validated successfully", slog.Int64("user_id", userID))
	return user, nil
}

// Refresh обновляет access-токен по refresh-токену
func (a *Auth) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	const op = "auth.Refresh"
	log := a.log.With(slog.String("op", op))

	// 1. Получить userID
	userID, err := a.refreshStorage.GetRefresh(ctx, refreshToken)
	if err != nil {
		log.Info("invalid refresh token", slog.String("error", err.Error()))
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// 2. Удалить старый
	_ = a.refreshStorage.DeleteRefresh(ctx, refreshToken)

	// Получаем пользователя, чтобы создать новый access-токен
	user, err := a.userStorage.UserByID(ctx, userID)
	if err != nil {
		log.Error("failed to get user by id", slog.String("error", err.Error()))
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	// 3. Создать новый access
	newAccess, err := jwt.NewToken(user, a.tokenTTL, a.secret)
	if err != nil {
		log.Error("failed to generate access token")
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// 4. Создать и сохранить новый refresh
	newRefresh := uuid.New().String()
	err = a.refreshStorage.SaveRefresh(ctx, newRefresh, userID, a.refreshTTL)
	if err != nil {
		log.Error("failed to save new refresh token")
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	log.Info("tokens refreshed successfully", slog.Int64("user_id", userID))
	return newAccess, newRefresh, nil
}

func (a *Auth) Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error {
	const op = "auth.Logout"

	log := a.log.With(
		slog.String("op", op),
		slog.String("jti", jti),
	)
	log.Info("logging out user, revoking token")

	// добавляем токен в хранилище отозванных
	a.tokenRevoker.Add(jti, exp)

	// удаляем refresh-токен, если он передан
	if refreshToken != "" {
		_ = a.refreshStorage.DeleteRefresh(ctx, refreshToken)
	}

	log.Info("token successfully revoked and refresh deleted")
	return nil
}

// RegisterNewUser создаёт нового пользователя с указанным email и паролем.
// Пароль хэшируется, и данные сохраняются через UserSaver. Возвращает
// идентификатор пользователя.
func (a *Auth) RegisterNewUser(ctx context.Context, email, password, name string) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)
	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash")
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.userStorage.SaveUser(ctx, email, passHash, name)
	if err != nil {
		if errors.Is(err, db.ErrUserExists) {
			log.Error("user already exists")
			return 0, fmt.Errorf("%s: %w", op, ErrUserAlreadyExists)
		}
		log.Error("failed to save user")
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user registered")
	return id, nil
}

// IsAdmin возвращает true, если пользователь с заданным ID обладает правами
// администратора.
func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)
	log.Info("checking if user is admin")

	isAdmin, err := a.userStorage.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("failed to check if user is admin")
		return false, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user is admin", slog.Bool("is_admin", isAdmin))
	return isAdmin, nil
}
