package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// HandleGetAds обрабатывает запросы на получение списка объявлений
// @Summary Получить список объявлений
// @Description Возвращает список всех объявлений
// @Tags ads
// @Produce json
// @Success 200 {array} models.Ad "список объявлений успешно получен"
// @Failure 400 {object} dto.ErrorResponse "Method not allowed: Метод не поддерживается (ожидается GET)"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера при получении объявлений"
// @Router /ads [get]
func (h *AdsHandlers) HandleGetAds(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		responser.RespondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	// копируем список объявлений без гонки данных
	adsList, err := h.services.Ads.GetAllAds(r.Context())
	if err != nil {
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// формируем и отправляем ответ
	responser.RespondWithJSON(w, http.StatusOK, adsList)
}

// HandleGetUserAds обрабатывает запросы на получение объявлений продавца по его ID
// @Summary Получить объявления пользователя
// @Description Возвращает список всех объявлений продавца по его ID
// @Tags ads
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string][]models.Ad "объявления успешно получены"
// @Failure 400 {object} map[string]string "некорректный ID пользователя"
// @Failure 500 {object} map[string]string "ошибка сервера"
// @Router /users/{id}/ads [get]
func (h *AdsHandlers) HandleGetUserAds(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleGetUserAds"

	// получаем ID из /api/v1/users/{id}/ads
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error(ErrInvalidUserID, slog.String("op", op), slog.String("id", idStr))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	// получаем список объявлений
	ads, err := h.services.Ads.GetAdsByUserID(r.Context(), userID)
	if err != nil {
		h.log.Error(ErrFailedToGetUserAds, slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToGetUserAds)
		return
	}

	// формируем ответ
	responser.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ads": ads,
	})
}
