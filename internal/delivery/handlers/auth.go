package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
)

// HandleRegister обрабатывает запросы на регистрацию новых пользователей
// @Summary Регистрация пользователя
// @Description Создаёт нового пользователя и автоматически выполняет вход
// @Tags auth
// @Accept json
// @Produce json
// @Param input body dto.RegisterRequest true "Registration data"
// @Success 200 {object} dto.LoginResponse "user registered and logged in successfully"
// @Failure 400 {object} dto.ErrorResponse "invalid request body / user already exists / validation failed (ValidationErrors): Ошибка формата запроса, дубликат пользователя или ошибка валидации"
// @Failure 400 {object} dto.ValidationErrors "Validation failed"
// @Failure 400 {object} dto.ErrorResponse "User already exists"
// @Failure 500 {object} dto.ErrorResponse "failed to register user / registered, but failed to login: Ошибка сервера при регистрации или авто-входе"
// @Router /auth/register [post]
func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	req.Name = sanitizer.StripHTML(req.Name)

	validationErrors := dto.ValidationErrors{}

	cleanEmail, err := validator.ValidateEmail(req.Email)
	if err != nil {
		validationErrors.Email = err.Error()
	}

	cleanName, err := validator.ValidateName(req.Name)
	if err != nil {
		validationErrors.Name = err.Error()
	}

	if err := validator.ValidatePassword(req.Password); err != nil {
		validationErrors.Password = err.Error()
	}

	if validationErrors.Email != "" || validationErrors.Password != "" || validationErrors.Name != "" {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	userID, err := h.services.Auth.RegisterNewUser(r.Context(), cleanEmail, req.Password, cleanName)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrUserAlreadyExists)
			return
		}

		h.log.ErrorContext(r.Context(), "failed to register user",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToRegisterUser)
		return
	}

	h.log.InfoContext(r.Context(), "user registered successfully",
		slog.Int64("user_id", userID),
	)

	token, refreshToken, user, err := h.services.Auth.Login(r.Context(), cleanEmail, req.Password)
	if err != nil {
		h.log.ErrorContext(r.Context(), "auto-login failed after registration",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrAutoLoginFailed)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	h.setAuthCookie(w, token, csrfToken)

	h.setRefreshCookie(w, refreshToken)
	h.respondWithUser(w, user)
}

// HandleLogin обрабатывает запросы на вход пользователя
// @Summary Вход пользователя
// @Description Аутентифицирует пользователя по email/пароль или, при наличии cookie, проверяет токен
// @Tags auth
// @Accept json
// @Produce json
// @Param input body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.LoginResponse "login successful"
// @Failure 400 {object} dto.ErrorResponse "invalid request body: Неверный формат тела запроса"
// @Failure 401 {object} dto.ErrorResponse "invalid credentials / invalid or expired token / email/password is required (ValidationErrors): Ошибка аутентификации или невалидный токен"
// @Failure 500 {object} dto.ErrorResponse "failed to login: Ошибка сервера при входе"
// @Router /auth/login [post]
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		h.handleTokenLogin(w, r, cookie.Value)
		return
	}

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	if req.Email == "" || req.Password == "" {
		validationErrors := dto.ValidationErrors{}
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
	validationErrors := dto.ValidationErrors{}

	cleanEmail, err := validator.ValidateEmail(email)
	if err != nil {
		validationErrors.Email = err.Error()
	}

	if err := validator.ValidatePassword(password); err != nil {
		validationErrors.Password = err.Error()
	}

	if validationErrors.Email != "" || validationErrors.Password != "" {
		h.log.WarnContext(r.Context(), "invalid login attempt",
			slog.String("email", email),
		)
		responser.RespondWithJSON(w, http.StatusUnauthorized, validationErrors)
		return
	}

	token, refreshToken, user, err := h.services.Auth.Login(r.Context(), cleanEmail, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			responser.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		h.log.ErrorContext(r.Context(), "failed to login",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogin)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, token, csrfToken)
	h.setRefreshCookie(w, refreshToken)
	h.respondWithUser(w, user)
}

// handleTokenLogin обрабатывает вход пользователя путём валидации существующего токена
func (h *AuthHandlers) handleTokenLogin(w http.ResponseWriter, r *http.Request, tokenString string) {
	user, err := h.services.Auth.ValidateTokenAndGetUser(r.Context(), tokenString)
	if err != nil {
		h.log.WarnContext(r.Context(), "invalid token attempt",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, tokenString, csrfToken)
	h.respondWithUser(w, user)
}

// HandleLogout обрабатывает запросы на выход из системы, добавляя jti токена в черный список.
// @Summary Выход пользователя
// @Description Инвалидирует текущую сессию и очищает аутентификационную куку
// @Tags auth
// @Success 200 {object} map[string]string "logout successful"
// @Failure 400 {object} dto.ErrorResponse "Missing CSRF cookie / CSRF token mismatch: Ошибка CSRF"
// @Failure 401 {object} dto.ErrorResponse "missing token cookie / invalid token / token has been revoked: Ошибка авторизации (Middleware) или internal error: Отсутствует jti"
// @Failure 500 {object} dto.ErrorResponse "failed to logout: Ошибка сервера при выходе"
// @Router /auth/logout [post]
// @Security CookieAuth
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	h.log.InfoContext(r.Context(), "logout attempt")

	jti, ok := r.Context().Value(middleware.JtiKey).(string)
	if !ok {
		h.log.ErrorContext(r.Context(), "jti not found in context")
		responser.RespondWithError(w, http.StatusUnauthorized, ErrInternalError)
		return
	}

	var refreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.services.Auth.Logout(r.Context(), jti, time.Now().Add(h.tokenTTL), refreshToken); err != nil {
		h.log.ErrorContext(r.Context(), "failed to logout in service",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogout)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleRefresh обновляет access токен по refresh токену
// @Summary Обновление токенов
// @Description Получает новый access и rottates refresh token
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string "tokens refreshed"
// @Failure 401 {object} dto.ErrorResponse "refresh token required / invalid refresh token: Ошибка refresh токена"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера при обновлении токенов"
// @Router /auth/refresh [post]
func (h *AuthHandlers) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		h.log.WarnContext(r.Context(), "refresh token missing")
		responser.RespondWithError(w, http.StatusUnauthorized, "refresh token required")
		return
	}

	newAccess, newRefresh, err := h.services.Auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to refresh tokens",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, newAccess, csrfToken)
	h.setRefreshCookie(w, newRefresh)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
