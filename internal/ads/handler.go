package ads

import (
	"ads/internal/utils"
	"net/http"
)

// Handler хранит зависимости для API
type Handler struct {
	repo *AdsRepository
}

// конструктор для обработчика
func NewHandler(r *AdsRepository) *Handler {
	return &Handler{repo: r}
}

// ручка для получения списка объявлений
func (h *Handler) GetAdsHandler(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		// формируем и отправляем ошибку 405
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// копируем список объявлений без гонки данных
	ads := h.repo.GetAll()

	// формируем и отправляем ответ
	utils.RespondWithJSON(w, http.StatusOK, ads)
}
