package handlers

import (
	"net/http"

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
