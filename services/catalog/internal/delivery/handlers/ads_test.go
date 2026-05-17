package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// createMultipartRequest создаёт multipart/form-data запрос с JSON-данными в поле "data"
func createMultipartRequest(ctx context.Context, method, url string, data interface{}) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if err := writer.WriteField("data", string(jsonData)); err != nil {
		return nil, err
	}
	_ = writer.Close()

	req := httptest.NewRequestWithContext(ctx, method, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// Тесты для обработчика получения объявлений

// Тест успешного выполнения
func TestGetAdsHandler_Success(t *testing.T) {
	// Получаем хендлер и мок напрямую из setupHandlers
	adsH, mockAds := setupAdsHandlers(t)

	testAds := []models.Ad{
		{ID: 1, Title: "Test Ad", Price: 100},
	}

	// Настраиваем ожидание мока
	mockAds.EXPECT().GetAllAds(gomock.Any()).Return(testAds, nil)

	// Создаем запрос (путь в данном случае не важен для прямого вызова метода)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ads", nil)
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
	adsH, _ := setupAdsHandlers(t)

	// Создаём POST запрос
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ads", nil)
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	adsH.HandleGetAds(rr, request)

	// Ожидаем код 400 (или 405, если логика внутри хендлера поменяется на MethodNotAllowed)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "expected status 400 for POST request")
}

// Проверка, что сервер не падает при отсутствии объявлений
func TestGetAdsHandler_EmptyData(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	// Возвращаем пустой слайс
	mockAds.EXPECT().GetAllAds(gomock.Any()).Return([]models.Ad{}, nil)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ads", nil)
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
	adsH, _ := setupAdsHandlers(t)

	// Создаём DELETE запрос
	request := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/ads", nil)
	rr := httptest.NewRecorder()

	adsH.HandleGetAds(rr, request)

	// Проверяем статус
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetUserAds_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(42)
	testAds := []models.Ad{
		{ID: 1, SellerID: userID, Title: "User's Ad", Price: 500},
	}

	mockAds.EXPECT().
		GetAdsByUserID(gomock.Any(), userID).
		Return(testAds, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}/ads", adsH.HandleGetUserAds)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/42/ads", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response struct {
		Ads []models.Ad `json:"ads"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testAds, response.Ads)
}

func TestHandleGetUserAds_InvalidID(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}/ads", adsH.HandleGetUserAds)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/abc/ads", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetUserAds_ServiceError(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(42)

	mockAds.EXPECT().
		GetAdsByUserID(gomock.Any(), userID).
		Return(nil, assert.AnError)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}/ads", adsH.HandleGetUserAds)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/42/ads", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleGetAdByID_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	adID := int64(1)
	testAd := models.Ad{ID: adID, Title: "Test Ad", Price: 100}

	mockAds.EXPECT().
		GetAdByID(gomock.Any(), adID).
		Return(testAd, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}", adsH.HandleGetAdByID)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ads/1", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualData models.Ad
	err := json.Unmarshal(rr.Body.Bytes(), &actualData)
	assert.NoError(t, err)
	assert.Equal(t, testAd, actualData)
}

func TestHandleGetAdByID_NotFound(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	adID := int64(1)

	mockAds.EXPECT().
		GetAdByID(gomock.Any(), adID).
		Return(models.Ad{}, ad.ErrAdNotFound)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}", adsH.HandleGetAdByID)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ads/1", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetAdByID_InvalidID(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}", adsH.HandleGetAdByID)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ads/abc", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreateAd_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	reqDto := dto.CreateAdRequest{
		Title:       "New Ad",
		Description: "Description",
		Price:       1000,
		CategoryID:  1,
		Status:      "active",
		Location:    "Moscow",
	}

	mockAds.EXPECT().
		CreateAd(gomock.Any(), gomock.Any()).
		Return(int64(123), nil)

	request, err := createMultipartRequest(t.Context(), http.MethodPost, "/ads", reqDto)
	assert.NoError(t, err)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	adsH.HandleCreateAd(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]int64
	_ = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, int64(123), response["ad_id"])
}

func TestHandleCreateAd_Unauthorized(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ads", nil)
	rr := httptest.NewRecorder()

	adsH.HandleCreateAd(rr, request)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleUpdateAdByID_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	adID := int64(10)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	title := "Updated Title"
	reqDto := dto.UpdateAdRequest{
		Title: &title,
	}

	mockAds.EXPECT().
		UpdateAd(gomock.Any(), gomock.Any()).
		Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /ads/{id}", adsH.HandleUpdateAdByID)

	request, err := createMultipartRequest(t.Context(), http.MethodPut, fmt.Sprintf("/ads/%d", adID), reqDto)
	assert.NoError(t, err)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleUpdateAdByID_Forbidden(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	title := "Title"
	reqDto := dto.UpdateAdRequest{Title: &title}

	mockAds.EXPECT().
		UpdateAd(gomock.Any(), gomock.Any()).
		Return(ad.ErrAdForbidden)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /ads/{id}", adsH.HandleUpdateAdByID)

	request, err := createMultipartRequest(t.Context(), http.MethodPut, "/ads/10", reqDto)
	assert.NoError(t, err)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestHandleDeleteAd_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	adID := int64(10)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mockAds.EXPECT().
		DeleteAd(gomock.Any(), adID, userID).
		Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /ads/{id}", adsH.HandleDeleteAd)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/ads/10", nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleCloseAdByID_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	adID := int64(10)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mockAds.EXPECT().
		CloseAd(gomock.Any(), adID, userID).
		Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /ads/{id}/close", adsH.HandleCloseAdByID)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ads/10/close", nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAddToFavorites_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	adID := int64(100)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mockAds.EXPECT().
		AddFavorite(gomock.Any(), userID, adID).
		Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /ads/{id}/favorite", adsH.HandleAddToFavorites)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, fmt.Sprintf("/ads/%d/favorite", adID), nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "ok", response["status"])
}

func TestHandleDeleteFromFavorites_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	adID := int64(100)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mockAds.EXPECT().
		RemoveFavorite(gomock.Any(), userID, adID).
		Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /ads/{id}/favorite", adsH.HandleDeleteFromFavorites)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, fmt.Sprintf("/ads/%d/favorite", adID), nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleGetFavorites_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	testAds := []models.Ad{
		{ID: 1, Title: "Favorite Ad 1", Price: 1000},
		{ID: 2, Title: "Favorite Ad 2", Price: 2000},
	}

	mockAds.EXPECT().
		GetUserFavorites(gomock.Any(), userID).
		Return(testAds, nil)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile/favorites", nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	adsH.HandleGetFavorites(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response struct {
		Ads []models.Ad `json:"ads"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testAds, response.Ads)
}

func TestHandleAddToFavorites_Unauthorized(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	// Контекст пустой, userID не положен
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ads/100/favorite", nil)
	rr := httptest.NewRecorder()

	adsH.HandleAddToFavorites(rr, request)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleAddToFavorites_InvalidAdID(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	userID := int64(1)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /ads/{id}/favorite", adsH.HandleAddToFavorites)

	// Передаем строку "abc" вместо ID
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ads/abc/favorite", nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetFavorites_ServiceError(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	userID := int64(1)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)

	mockAds.EXPECT().
		GetUserFavorites(gomock.Any(), userID).
		Return(nil, assert.AnError)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile/favorites", nil)
	request = request.WithContext(ctx)
	rr := httptest.NewRecorder()

	adsH.HandleGetFavorites(rr, request)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleGetCategoryCharacteristics_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	categoryID := int64(10)
	expected := []models.CategoryCharacteristic{
		{ID: 1, CategoryID: categoryID, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}, SortOrder: 1},
		{ID: 2, CategoryID: categoryID, Name: "Размер", AllowedValues: nil, SortOrder: 2},
	}

	mockAds.EXPECT().
		GetCategoryCharacteristics(gomock.Any(), categoryID).
		Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /categories/{id}/characteristics", adsH.HandleGetCategoryCharacteristics)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories/10/characteristics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualData []models.CategoryCharacteristic
	err := json.Unmarshal(rr.Body.Bytes(), &actualData)
	assert.NoError(t, err)
	assert.Equal(t, expected, actualData)
}

func TestHandleGetCategoryCharacteristics_InvalidID(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /categories/{id}/characteristics", adsH.HandleGetCategoryCharacteristics)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories/abc/characteristics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetCategoryCharacteristics_ServiceError(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	categoryID := int64(10)

	mockAds.EXPECT().
		GetCategoryCharacteristics(gomock.Any(), categoryID).
		Return(nil, assert.AnError)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /categories/{id}/characteristics", adsH.HandleGetCategoryCharacteristics)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories/10/characteristics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleGetPriceHistory_Success(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	adID := int64(1)
	want := []models.PricePoint{{Price: 15000}, {Price: 12000}}

	mockAds.EXPECT().
		GetPriceHistory(gomock.Any(), adID).
		Return(want, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}/price-history", adsH.HandleGetPriceHistory)

	request := httptest.NewRequest(http.MethodGet, "/ads/1/price-history", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp dto.PriceHistoryResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, want, resp.History)
}

func TestHandleGetPriceHistory_NotFound(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	mockAds.EXPECT().
		GetPriceHistory(gomock.Any(), int64(1)).
		Return(nil, ad.ErrAdNotFound)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}/price-history", adsH.HandleGetPriceHistory)

	request := httptest.NewRequest(http.MethodGet, "/ads/1/price-history", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetPriceHistory_InvalidID(t *testing.T) {
	adsH, _ := setupAdsHandlers(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}/price-history", adsH.HandleGetPriceHistory)

	request := httptest.NewRequest(http.MethodGet, "/ads/abc/price-history", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetPriceHistory_InternalError(t *testing.T) {
	adsH, mockAds := setupAdsHandlers(t)

	mockAds.EXPECT().
		GetPriceHistory(gomock.Any(), int64(1)).
		Return(nil, fmt.Errorf("db error"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ads/{id}/price-history", adsH.HandleGetPriceHistory)

	request := httptest.NewRequest(http.MethodGet, "/ads/1/price-history", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
