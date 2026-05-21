package handlers

// HTTP-обработчики для Auth-сервиса.
// Shared-пакеты (validator, responser, sanitizer, middleware) импортируются
// из корневого pkg/ — единый go.mod позволяет это без дублирования.

import (
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
)

// Константы HTTP-слоя: лимиты, имена cookie и сообщения об ошибках.
const (
	MaxUploadSize = 5 << 20

	cookieNameToken = "token"
	cookieNameCSRF  = "csrf_token"

	ErrInvalidRequestBody   = "invalid request body"
	ErrUserAlreadyExists    = "user already exists"
	ErrFailedToRegisterUser = "failed to register user"
	ErrAutoLoginFailed      = "registered, but failed to login"
	ErrFailedToLogin        = "failed to login"
	ErrInternalError        = "internal error"
	ErrFailedToLogout       = "failed to logout"
	ErrInvalidUserID        = "invalid user id"
	ErrUnauthorized         = "unauthorized"
	ErrFileTooBig           = "file too big"
	ErrFailedToGetFile      = "failed to get file"

	statusField = "status"
	statusOK    = "ok"
)

// Auth описывает методы сервиса аутентификации
type Auth interface {
	Login(ctx context.Context, email string, password string) (string, string, models.User, error)
	ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error)
	RegisterNewUser(ctx context.Context, email string, password string, name string) (userID int64, err error)
	Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	GetProfile(ctx context.Context, userID int64) (models.User, error)
	UpdateProfile(ctx context.Context, userID int64, name string) (models.User, error)
	UpdateAvatar(ctx context.Context, userID int64, file multipart.File, filename string) (models.User, error)
}

// AuthHandlers содержит обработчики для аутентификации
type AuthHandlers struct {
	log        *slog.Logger
	auth       Auth
	tokenTTL   time.Duration
	refreshTTL time.Duration
	secret     string
}

// NewAuthHandlers создает новый экземпляр AuthHandlers
func NewAuthHandlers(log *slog.Logger, auth Auth, tokenTTL time.Duration, refreshTTL time.Duration, secret string) *AuthHandlers {
	return &AuthHandlers{
		log:        log,
		auth:       auth,
		tokenTTL:   tokenTTL,
		refreshTTL: refreshTTL,
		secret:     secret,
	}
}

// setAuthCookie устанавливает cookie с JWT-токеном и CSRF-токеном
func (h *AuthHandlers) setAuthCookie(w http.ResponseWriter, token string, csrfToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameToken,
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.tokenTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameCSRF,
		Value:    csrfToken,
		HttpOnly: false,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.tokenTTL.Seconds()),
	})
}

// setRefreshCookie устанавливает cookie с refresh-токеном
func (h *AuthHandlers) setRefreshCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.refreshTTL.Seconds()),
	})
}

// respondWithUser отправляет успешный ответ с данными пользователя
func (h *AuthHandlers) respondWithUser(w http.ResponseWriter, user models.User) {
	responser.RespondWithJSON(w, http.StatusOK, dto.LoginResponse{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
	})
}
