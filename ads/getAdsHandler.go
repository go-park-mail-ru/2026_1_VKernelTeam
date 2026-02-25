package main

import (
	"encoding/json"
	"net/http"
)

// структура объявления
type Ad struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int    `json:"price"`
}

// ручка для получения объявлений
func getAdsHandler(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		// ошибка 405 - method not allowed
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// устанавливаем заголовок, что возвращаем JSON
	w.Header().Set("Content-Type", "application/json")

	// код 200 возвращается по умолчанию
	// w.WriteHeader(http.StatusOK)

	// кодируем срез объявлений в JSON и отправляем в ответ
	err := json.NewEncoder(w).Encode(ads)
	if err != nil {
		// если вдруг JSON не смог собраться, отдаем 500
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
