package main

import (
	"net/http"
	"time"
)

// структура объявления
type Ad struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Photos      []string  `json:"photos"`
	Tags        []string  `json:"tags"`
	SellerID    int       `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
	Views       int       `json:"views"`
}

// ручка для получения объявлений
func getAdsHandler(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		// формируем и отправляем ошибку 405
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// копируем список объявлений без гонки данных
	repo.RLock()
	ads := repo.data
	repo.RUnlock()

	// формируем и отправляем ответ
	respondWithJSON(w, http.StatusOK, ads)
}
