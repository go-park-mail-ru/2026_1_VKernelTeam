package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
)

// HandleLogin обрабатывает запросы на вход пользователя
// @Summary Вход пользователя
// @Description Аутентифицирует пользователя по email/пароль или, при наличии cookie, проверяет токен
// @Tags auth
// @Accept json
// @Produce json
// @Param input body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "login successful"
// @Failure 400 {object} ErrorResponse "invalid request body or missing fields"
// @Failure 401 {object} ValidationErrors "email/password validation errors or invalid credentials/token"
// @Failure 500 {object} ErrorResponse "internal server error"
// @Router /auth/login [post]
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// сначала ищем токен в куке
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		// есть токен, пытаемся его валидировать
		h.handleTokenLogin(w, r, cookie.Value)
		return
	}

	// иначе - вход по email/пароль из тела
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	if req.Email == "" || req.Password == "" {
		validationErrors := ValidationErrors{}
		if req.Email == "" {
			validationErrors.Email = "email is required"
		}
		if req.Password == "" {
			validationErrors.Password = "password is required"
		}
		responser.RespondWithJSON(w, http.StatusUnauthorized, validationErrors)
		return
	}

	h.handleCredentialsLogin(w, r, req.Email, req.Password)
}

// handleCredentialsLogin обрабатывает вход пользователя по email и паролю
func (h *AuthHandlers) handleCredentialsLogin(w http.ResponseWriter, r *http.Request, email, password string) {
	validationErrors := ValidationErrors{}
	if err := validator.ValidateEmail(email); err != nil {
		validationErrors.Email = err.Error()
	}
	if err := validator.ValidatePassword(password); err != nil {
		validationErrors.Password = err.Error()
	}

	// Если есть хотя бы одна ошибка валидации
	if validationErrors.Email != "" || validationErrors.Password != "" {
		h.log.Info(
			"invalid login attempt",
			slog.String("email", email),
			slog.String("email_error", validationErrors.Email),
			slog.String("password_error", validationErrors.Password),
		)
		responser.RespondWithJSON(w, http.StatusUnauthorized, validationErrors)
		return
	}

	token, refreshToken, user, err := h.services.Auth.Login(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			responser.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		h.log.Error("failed to login", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogin)
		return
	}

	h.setAuthCookie(w, token)
	h.setRefreshCookie(w, refreshToken)
	h.respondWithUser(w, user)
}

// handleTokenLogin обрабатывает вход пользователя путём валидации существующего токена
func (h *AuthHandlers) handleTokenLogin(w http.ResponseWriter, r *http.Request, tokenString string) {
	user, err := h.services.Auth.ValidateTokenAndGetUser(r.Context(), tokenString)
	if err != nil {
		h.log.Info("invalid token attempt", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	// Устанавливаем куку с токеном
	h.setAuthCookie(w, tokenString)
	h.respondWithUser(w, user)
}
