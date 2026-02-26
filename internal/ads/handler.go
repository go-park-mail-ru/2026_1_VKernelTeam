package ads

import (
	"ads/internal/utils"
	"net/http"
)

// ручка для получения списка объявлений
func GetAdsHandler(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		// формируем и отправляем ошибку 405
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// копируем список объявлений без гонки данных
	Repo.RLock()
	ads := Repo.data
	Repo.RUnlock()

	// формируем и отправляем ответ
	utils.RespondWithJSON(w, http.StatusOK, ads)
}
