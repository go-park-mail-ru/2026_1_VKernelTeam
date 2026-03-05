package responser

import (
	"encoding/json"
	"net/http"
)

// RespondWithJSON отправляет готовый объект
func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	// преобразуем полученные данные в json
	response, err := json.Marshal(payload)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "couldn't convert the received data to JSON")
		return
	}

	// устанавливаем заголовок, что возвращаем JSON
	w.Header().Set("Content-Type", "application/json")

	// устанавливаем код ответа
	w.WriteHeader(code)

	// записываем данные
	w.Write(response)
}

// RespondWithError отправляет структурированную ошибку
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}
