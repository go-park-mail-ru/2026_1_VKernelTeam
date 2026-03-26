package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// Тесты для обработчика получения объявлений

// Тест успешного выполнения
func TestGetAdsHandler_Success(t *testing.T) {
	// Получаем хендлер и мок напрямую из setupHandlers
	_, adsH, _, mockAds := setupHandlers(t)

	testAds := []models.Ad{
		{ID: 1, Title: "Test Ad", Price: 100},
	}

	// Настраиваем ожидание мока
	mockAds.EXPECT().GetAllAds(gomock.Any()).Return(testAds, nil)

	// Создаем запрос (путь в данном случае не важен для прямого вызова метода)
	request := httptest.NewRequest(http.MethodGet, "/ads", nil)
	rr := httptest.NewRecorder()

	// Вызываем метод хендлера напрямую
	adsH.HandleGetAds(rr, request)

	// Проверяем результат
	assert.Equal(t, http.StatusOK, rr.Code)

	var actualData []models.Ad
	err := json.Unmarshal(rr.Body.Bytes(), &actualData)
	assert.NoError(t, err)
	assert.Equal(t, testAds, actualData)
}

// Проверка ограничения методов (обрабатываем только GET)
func TestGetAdsHandler_OnlyGet(t *testing.T) {
	_, adsH, _, _ := setupHandlers(t)

	// Создаём POST запрос
	request := httptest.NewRequest(http.MethodPost, "/ads", nil)
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	adsH.HandleGetAds(rr, request)

	// Ожидаем код 400 (или 405, если логика внутри хендлера поменяется на MethodNotAllowed)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "expected status 400 for POST request")
}

// Проверка, что сервер не падает при отсутствии объявлений
func TestGetAdsHandler_EmptyData(t *testing.T) {
	_, adsH, _, mockAds := setupHandlers(t)

	// Возвращаем пустой слайс
	mockAds.EXPECT().GetAllAds(gomock.Any()).Return([]models.Ad{}, nil)

	request := httptest.NewRequest(http.MethodGet, "/ads", nil)
	rr := httptest.NewRecorder()

	adsH.HandleGetAds(rr, request)

	// Проверяем статус-код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Получаем тело ответа
	var actualData []models.Ad
	err := json.Unmarshal(rr.Body.Bytes(), &actualData)
	assert.NoError(t, err, "failed to decode JSON")

	// Проверяем, что вернулся пустой массив (не nil)
	assert.NotNil(t, actualData)
	assert.Len(t, actualData, 0)
}

// Тестируем ошибку метода (дублирует логику OnlyGet, но для консистентности)
func TestGetAdsHandler_WrongMethod(t *testing.T) {
	_, adsH, _, _ := setupHandlers(t)

	// Создаём DELETE запрос
	request := httptest.NewRequest(http.MethodDelete, "/ads", nil)
	rr := httptest.NewRecorder()

	adsH.HandleGetAds(rr, request)

	// Проверяем статус
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
