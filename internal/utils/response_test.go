package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// тест получения ошибки преобразования переданного параметра в JSON
func TestRespondWithJSON_Error(t *testing.T) {
	rr := httptest.NewRecorder()

	// канал нельзя преобразовать в JSON
	invalidData := make(chan int)

	RespondWithJSON(rr, http.StatusOK, invalidData)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 error, got %d", rr.Code)
	}
}
