package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
// @Param input body RegisterRequest true "Registration data"
// @Success 200 {object} LoginResponse "user registered and logged in successfully"
// @Failure 400 {object} ErrorResponse "invalid request body / user already exists / validation failed (ValidationErrors): Ошибка формата запроса, дубликат пользователя или ошибка валидации"
// @Failure 500 {object} ErrorResponse "failed to register user / registered, but failed to login: Ошибка сервера при регистрации или авто-входе"
// @Router /auth/register [post]
func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// Собираем все ошибки валидации
	validationErrors := ValidationErrors{}
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
	h.respondWithUser(w, user, csrfToken)
}
