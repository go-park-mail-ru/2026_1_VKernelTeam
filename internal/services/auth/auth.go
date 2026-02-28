// Package auth реализует бизнес-логику аутентификации и авторизации
// пользователей. Он определяет сервис Auth с методами входа в систему,
// регистрации и проверки прав администратора, а также соответствующие
// интерфейсы для взаимодействия с хранилищем и провайдерами данных.
// Пакет auth реализует бизнес-логику аутентификации и авторизации
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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/lib/jwt"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

// Auth представляет собой сервис аутентификации. Он использует логгер,
// провайдеров пользователей и приложений, а также TTL для генерируемых
// токенов.
type Auth struct {
	log          *slog.Logger
	userSaver    UserSaver
	userProvider UserProvider
	appProvider  AppProvider
	tokenTTL     time.Duration
}

// UserSaver описывает интерфейс для сохранения нового пользователя
// в хранилище. Реализация должна возвращать идентификатор и ошибку.
type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (uid int64, err error)
}

// UserProvider предоставляет методы получения данных о пользователе и
// проверки его административных прав.
type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// AppProvider отвечает за получение информации о зарегистрированных
// приложениях.
type AppProvider interface {
	App(ctx context.Context, appID int64) (models.App, error)
}

// ErrInvalidCredentials возвращается, когда email/пароль не совпадают с
// сохранёнными данными.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// New создаёт новый экземпляр Auth с переданными зависимостями.
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		userSaver:    userSaver,
		userProvider: userProvider,
		appProvider:  appProvider,
		tokenTTL:     tokenTTL,
	}
}

// New создаёт новый экземпляр Auth с переданными зависимостями.
func (a *Auth) Login(ctx context.Context, email, password string, appId int64) (string, error) {
	const op = "auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)
	log.Info("logging in user")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error("user not found")
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		log.Error("failed to get user")
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials")
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to generate token")
		return "", fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user logged in")
	return token, nil
}

// Login аутентифицирует пользователя по email и паролю, проверяет
// принадлежность к приложению и возвращает JWT-токен. В случае
// ошибок возвращается описанная ошибка.
func (a *Auth) RegisterNewUser(ctx context.Context, email, password string) (int64, error) {
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

	id, err := a.userSaver.SaveUser(ctx, email, passHash)
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

// RegisterNewUser создаёт нового пользователя с указанным email и паролем.
// Пароль хэшируется, и данные сохраняются через UserSaver. Возвращает
// идентификатор пользователя.
func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)
	log.Info("checking if user is admin")

	isAdmin, err := a.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("failed to check if user is admin")
		return false, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user is admin", slog.Bool("is_admin", isAdmin))
	return isAdmin, nil
}

// IsAdmin возвращает true, если пользователь с заданным ID обладает правами
// администратора.
