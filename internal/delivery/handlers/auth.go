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
// @Failure 500 {object} dto.ErrorResponse "failed to register user / registered, but failed to login: Ошибка сервера при регистрации или авто-входе"
// @Router /auth/register [post]
func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// Собираем все ошибки валидации
	validationErrors := dto.ValidationErrors{}
	if err := validator.ValidateEmail(req.Email); err != nil {
		validationErrors.Email = err.Error()
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		validationErrors.Password = err.Error()
	}
	if err := validator.ValidateName(req.Name); err != nil {
		validationErrors.Name = err.Error()
	}

	// Если есть хотя бы одна ошибка валидации, возвращаем их все
	if validationErrors.Email != "" || validationErrors.Password != "" || validationErrors.Name != "" {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	userID, err := h.services.Auth.RegisterNewUser(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrUserAlreadyExists)
			return
		}

		h.log.Error("failed to register user", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToRegisterUser)
		return
	}

	h.log.Info(
		"user registered successfully",
		slog.Int64("user_id", userID),
		slog.String("email", req.Email),
	)

	// сразу логиним
	token, refreshToken, user, err := h.services.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.log.Error("auto-login failed after registration", slog.String("error", err.Error()))
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
	// сначала ищем токен в куке
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		// есть токен, пытаемся его валидировать
		h.handleTokenLogin(w, r, cookie.Value)
		return
	}

	// иначе - вход по email/пароль из тела
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

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, token, csrfToken)
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

	csrfToken := middleware.GenerateCSRFToken()

	// Устанавливаем куку с токенами
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
// @Security CsrfCookieAuth
// @Security CsrfHeaderAuth
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	h.log.Info("logout attempt", slog.String("op", "HandleLogout"))

	// достаём jti из контекста
	jti, ok := r.Context().Value(middleware.JtiKey).(string)
	if !ok {
		h.log.Error("jti not found in context")
		responser.RespondWithError(w, http.StatusUnauthorized, ErrInternalError)
		return
	}

	var refreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.services.Auth.Logout(r.Context(), jti, time.Now().Add(h.tokenTTL), refreshToken); err != nil {
		h.log.Error("failed to logout in service", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogout)
		return
	}

	// удаляем JWT
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	// удаляем CSRF
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
		h.log.Info("refresh token missing")
		responser.RespondWithError(w, http.StatusUnauthorized, "refresh token required")
		return
	}

	newAccess, newRefresh, err := h.services.Auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.log.Error("failed to refresh tokens", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, newAccess, csrfToken)
	h.setRefreshCookie(w, newRefresh)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
