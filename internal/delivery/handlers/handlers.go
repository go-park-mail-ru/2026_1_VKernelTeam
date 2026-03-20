package handlers

// Package handlers содержит HTTP-обработчики для всех маршрутов приложения
import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
)

// ошибки HTTP-обработчиков
const (
	ErrInvalidRequestBody   = "invalid request body"
	ErrUserAlreadyExists    = "user already exists"
	ErrFailedToRegisterUser = "failed to register user"
	ErrAutoLoginFailed      = "registered, but failed to login"
	ErrFailedToLogin        = "failed to login"
	ErrInternalError        = "internal error"
	ErrFailedToLogout       = "failed to logout"
	ErrMethodNotAllowed     = "Method not allowed"
)

// Services объединяет все бизнес-сервисы приложения, необходимые хендлерам
type Services struct {
	Ads  Ads
	Auth Auth
}

// Ads описывает методы сервиса объявлений
type Ads interface {
	GetAll(ctx context.Context) ([]models.Ad, error)
}

// Auth описывает минимальный набор методов сервиса аутентификации
type Auth interface {
	Login(ctx context.Context, email string, password string) (string, models.User, error)
	ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error)
	RegisterNewUser(ctx context.Context, email string, password string, name string) (userID int64, err error)
	Logout(ctx context.Context, jti string, exp time.Time) error
}

// AuthHandlers содержит обработчики для аутентификации
type AuthHandlers struct {
	log       *slog.Logger
	services  Services
	blacklist auth.TokenRevoker
	tokenTTL  time.Duration
	secret    string
}

// AdsHandlers содержит обработчики для объявлений
type AdsHandlers struct {
	log      *slog.Logger
	services Services
}

// NewAuthHandlers создает новый экземпляр AuthHandlers
func NewAuthHandlers(log *slog.Logger, services Services, bl auth.TokenRevoker, tokenTTL time.Duration, secret string) *AuthHandlers {
	return &AuthHandlers{
		log:       log,
		services:  services,
		blacklist: bl,
		tokenTTL:  tokenTTL,
		secret:    secret,
	}
}

// NewAdsHandlers создает новый экземпляр AdsHandlers
func NewAdsHandlers(log *slog.Logger, services Services) *AdsHandlers {
	return &AdsHandlers{
		log:      log,
		services: services,
	}
}

// RegisterRequest представляет собой структуру для запроса на регистрацию пользователя
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest представляет собой структуру для запроса на вход в систему
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse представляет собой структуру для ответа на запрос входа в систему
type LoginResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// ErrorResponse представляет собой структуру для отправки ошибок в формате JSON
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors представляет собой структуру для отправки ошибок валидации по полям
type ValidationErrors struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

// setAuthCookie устанавливает cookie с токеном
func (h *AuthHandlers) setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.tokenTTL.Seconds()),
	})
}

// respondWithUser отправляет успешный ответ с данными пользователя
func (h *AuthHandlers) respondWithUser(w http.ResponseWriter, user models.User) {
	responser.RespondWithJSON(w, http.StatusOK, LoginResponse{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	})
}
