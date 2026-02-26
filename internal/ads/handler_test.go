package ads

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// тест успешного выполнения
func TestGetAdsHandler_Success(t *testing.T) {
	// создаём чистые зависимости для теста
	repo := NewAdsRepository()
	handler := NewHandler(repo)

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("GET", "/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку дял ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	handler.GetAdsHandler(rr, request)

	// проверяем статус-код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf(
			"handler returned wrong status code: got %v want %v",
			status,
			http.StatusOK,
		)
	}

	// првоеряем тип контента
	expectedType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedType {
		t.Errorf(
			"handler returned unexpected content type: got %v want %v",
			contentType,
			expectedType,
		)
	}
}

// првоерка ограничения методов (обрабатываем только GET)
func TestGetAdsHandler_OnlyGet(t *testing.T) {
	// создаём чистые зависимости для теста
	repo := NewAdsRepository()
	handler := NewHandler(repo)

	// создаём POST запрос к эндпоинту
	request, err := http.NewRequest("POST", "/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку дял ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	handler.GetAdsHandler(rr, request)

	// ожидаем код 405 - method not allowed
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST request, got %v", rr.Code)
	}
}

// првоерка, что сервер не падает при отсутствии объявлений
func TestGetAdsHandler_EmptyData(t *testing.T) {
	repo := &AdsRepository{data: []Ad{}}
	handler := NewHandler(repo)

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("GET", "/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку дял ответа
	rr := httptest.NewRecorder()

	// вызываем обработчик
	handler.GetAdsHandler(rr, request)

	// проверяем статус-код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 even with empty data, got %v", status)
	}

	// првоеряем, что вернулся пустой массив, а не null
	if rr.Body.String() == "null\n" {
		t.Error("handler returned null instead of empty array []")
	}
}

// тестируем ошибку сервера
func TestGetAdsHandler_WrongMethod(t *testing.T) {
	// создаём чистые зависимости для теста
	repo := NewAdsRepository()
	handler := NewHandler(repo)

	// создаём запрос к эндпоинту
	request, err := http.NewRequest("POST", "/ads", nil)
	if err != nil {
		t.Fatal(err)
	}

	// создаём RequestRecoder - заглушку для ответа
	rr := httptest.NewRecorder()

	// вызываем handler
	handler.GetAdsHandler(rr, request)

	// проверяем статус
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}
