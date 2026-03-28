package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// HandleGetProfile возвращает профиль текущего авторизованного пользователя
// @Summary Получить профиль пользователя
// @Description Возвращает данные профиля на основе userID из контекста (JWT)
// @Tags auth
// @Produce json
// @Success 200 {object} models.User "профиль успешно получен"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: внутренняя ошибка сервера"
// @Security CookieAuth
// @Router /profile [get]
func (h *AuthHandlers) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleGetProfile"

	// извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// получаем профиль
	user, err := h.services.Auth.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get profile", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, user)
}

// HandleUpdateProfile обновляет данные профиля текущего пользователя
// @Summary Обновить профиль
// @Description Обновляет имя пользователя. Требует действующую сессию и валидный CSRF-токен.
// @Tags auth
// @Accept json
// @Produce json
// @Param input body dto.UpdateProfileRequest true "Новые данные профиля"
// @Success 200 {object} models.User "профиль успешно обновлен"
// @Failure 400 {object} dto.ErrorResponse "invalid request body / Missing CSRF cookie / CSRF token mismatch"
// @Failure 401 {object} dto.ErrorResponse "missing token cookie / invalid token / token has been revoked"
// @Failure 500 {object} dto.ErrorResponse "internal error: внутренняя ошибка сервера"
// @Security CookieAuth
// @Security CsrfHeaderAuth
// @Router /profile/update [post]
func (h *AuthHandlers) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleUpdateProfile"

	// извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// декодируем тело запроса
	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode request body", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// обновляем профиль
	updatedUser, err := h.services.Auth.UpdateProfile(r.Context(), userID, req.Name)
	if err != nil {
		h.log.Error("failed to update profile", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, updatedUser)
}

// HandleGetPublicProfile возвращает публичные данные любого пользователя по ID
// @Summary Получить публичный профиль
// @Description Возвращает только общедоступную информацию: имя, рейтинг, кол-во объявлений. Не требует авторизации.
// @Tags auth
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} dto.PublicUserResponse "публичный профиль успешно получен"
// @Failure 400 {object} dto.ErrorResponse "invalid user id / user not found"
// @Failure 500 {object} dto.ErrorResponse "internal error: внутренняя ошибка сервера"
// @Router /users/{id} [get]
func (h *AuthHandlers) HandleGetPublicProfile(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleGetPublicProfile"

	// извлекаем userID из пути
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error("invalid user id", slog.String("op", op), slog.String("id", idStr))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	// получаем профиль
	user, err := h.services.Auth.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get profile", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// формирует ответ
	response := dto.PublicUserResponse{
		ID:           user.ID,
		Name:         user.Name,
		AvatarPath:   user.AvatarPath,
		Rating:       user.Rating,
		ReviewsCount: user.ReviewsCount,
		AdsCount:     user.AdsCount,
		CreatedAt:    user.CreatedAt,
	}

	responser.RespondWithJSON(w, http.StatusOK, response)
}

// TODO
func (h *AuthHandlers) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
