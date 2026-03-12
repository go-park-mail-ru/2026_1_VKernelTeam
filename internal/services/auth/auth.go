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
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/lib/jwt"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

// TokenRevoker описывает интерфейс для отзыва токенов
type TokenRevoker interface {
	Add(jti string, exp time.Time)
}

// UserProviderSaver предоставляет методы сохранения пользователя в хранилище
// получения данных о пользователе и проверки его административных прав.
type UserProviderSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte, name string) (uid int64, err error)
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// Auth представляет собой сервис аутентификации. Он использует логгер,
// провайдеров пользователей и приложений, а также TTL для генерируемых
// токенов.
type Auth struct {
	log          *slog.Logger
	userStorage  UserProviderSaver
	tokenRevoker TokenRevoker
	tokenTTL     time.Duration
	secret       string
}

// ErrInvalidCredentials возвращается, когда email/пароль не совпадают с
// сохранёнными данными.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// New создаёт новый экземпляр Auth с переданными зависимостями.
func New(
	log *slog.Logger,
	userStorage UserProviderSaver,
	tokenRevoker TokenRevoker,
	tokenTTL time.Duration,
	secret string,
) *Auth {
	return &Auth{
		log:          log,
		userStorage:  userStorage,
		tokenRevoker: tokenRevoker,
		tokenTTL:     tokenTTL,
		secret:       secret,
	}
}

// Login аутентифицирует пользователя по email и паролю и возвращает JWT-токен. В случае
// ошибок возвращается описанная ошибка.
func (a *Auth) Login(ctx context.Context, email, password string) (string, models.User, error) {

	log := a.log.With(slog.String("email", email))
	log.Info("logging in user")

	user, err := a.userStorage.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error("user not found")
			return "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
		}
		log.Error("failed to get user")
		return "", models.User{}, fmt.Errorf("%w", err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials")
		return "", models.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
	}

	token, err := jwt.NewToken(user, a.tokenTTL, a.secret)
	if err != nil {
		log.Error("failed to generate token")
		return "", models.User{}, fmt.Errorf("%w", err)
	}

	log.Info("user logged in")
	return token, user, nil
}

// Logout отзывает токен пользователя, добавляя в чёрный список его jti
func (a *Auth) Logout(ctx context.Context, jti string, exp time.Time) error {
	const op = "auth.Logout"

	log := a.log.With(
		slog.String("op", op),
		slog.String("jti", jti),
	)
	log.Info("logging out user, revoking token")

	// добавляем токен в хранилище отозванных
	a.tokenRevoker.Add(jti, exp)

	log.Info("token successfully revoked")
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
		if errors.Is(err, storage.ErrUserExists) {
			log.Error("user already exists")
			return 0, fmt.Errorf("%s: %w", op, err)
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
		if errors.Is(err, storage.ErrUserNotFound) {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("failed to check if user is admin")
		return false, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user is admin", slog.Bool("is_admin", isAdmin))
	return isAdmin, nil
}
