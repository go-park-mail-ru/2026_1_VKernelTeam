package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
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

	// Извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// Получаем профиль
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

	// Извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// Декодируем тело запроса
	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode request body", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// Удаляем HTML-теги из имени (защита от XSS)
	req.Name = sanitizer.StripHTML(req.Name)

	// Валидируем имя
	cleanName, err := validator.ValidateName(req.Name)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Обновляем профиль
	updatedUser, err := h.services.Auth.UpdateProfile(r.Context(), userID, cleanName)
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

	// Извлекаем userID из пути
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error("invalid user id", slog.String("op", op), slog.String("id", idStr))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	// Получаем профиль
	user, err := h.services.Auth.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get profile", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// Формирует ответ
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

// HandleUploadAvatar загружает аватарку пользователя (multipart/form-data)
// @Summary Загрузить аватар
// @Description Принимает файл через multipart/form-data. Поле: "avatar"
// @Tags auth
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Файл изображения"
// @Success 200 {object} models.User "аватар обновлен"
// @Failure 400 {object} dto.ErrorResponse "file too big / failed to get file"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error: внутренняя ошибка сервера"
// @Router /profile/avatar [post]
func (h *AuthHandlers) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleUploadAvatar"

	h.log.Debug("checking content type", slog.String("ct", r.Header.Get("Content-Type")))

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// Ограничиваем размер аватарки
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		h.log.Error("parse multipart form error", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		h.log.Error("failed to get file", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToGetFile)
		return
	}
	defer file.Close()

	// Передаем в Usecase
	updatedUser, err := h.services.Auth.UpdateAvatar(r.Context(), userID, file, header.Filename)
	if err != nil {
		h.log.Error("failed to update avatar", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, updatedUser)
}
