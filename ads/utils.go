package main

import (
	"encoding/json"
	"net/http"
)

// respondWithJSON отправляет готовый объект
func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	// преобразуем полученные данные в json
	response, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't convert the received data to JSON")
		return
	}

	// устанавливаем заголовок, что возвращаем JSON
	w.Header().Set("Content-Type", "application/json")

	// устанавливаем код ответа
	w.WriteHeader(code)

	// записываем данные
	w.Write(response)
}

// respondWithError отправляет структурированную ошибку
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
