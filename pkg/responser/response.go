package responser

import (
	"encoding/json"
	"net/http"

	"github.com/mailru/easyjson"
)

var (
	// ErrJSONMarshalFailed возвращается клиенту, когда сериализация ответа в JSON завершилась ошибкой.
	ErrJSONMarshalFailed = "couldn't convert the received data to JSON"
)

// RespondWithJSON отправляет готовый объект.
// Если payload реализует easyjson.Marshaler — используется быстрая сериализация
// без reflection; иначе fallback на encoding/json (для map и анонимных структур).
func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	var (
		response []byte
		err      error
	)

	if m, ok := payload.(easyjson.Marshaler); ok {
		response, err = easyjson.Marshal(m)
	} else {
		response, err = json.Marshal(payload)
	}
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, ErrJSONMarshalFailed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

// RespondWithError отправляет структурированную ошибку
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}
