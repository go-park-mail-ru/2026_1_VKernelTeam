package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// mockAdsService реализует интерфейс Ads для тестов
type mockAdsService struct {
	GetAllFunc func() []models.Ad
}

func (m *mockAdsService) GetAll(_ context.Context) ([]models.Ad, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(), nil
	}
	return []models.Ad{}, nil
}

func newTestAdsHandlers(adsSvc Ads) *AdsHandlers {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewAdsHandlers(log, Services{Ads: adsSvc})
}

// TestHandleGetAds_Success — GET-запрос возвращает список объявлений с кодом 200
func TestHandleGetAds_Success(t *testing.T) {
	ads := []models.Ad{
		{ID: 1, Title: "First Ad", Price: 1000},
		{ID: 2, Title: "Second Ad", Price: 2000},
	}
	h := newTestAdsHandlers(&mockAdsService{
		GetAllFunc: func() []models.Ad { return ads },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ads", nil)
	w := httptest.NewRecorder()

	h.HandleGetAds(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result []models.Ad
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != len(ads) {
		t.Errorf("expected %d ads, got %d", len(ads), len(result))
	}
	if result[0].ID != ads[0].ID {
		t.Errorf("expected first ad ID %d, got %d", ads[0].ID, result[0].ID)
	}
}

// TestHandleGetAds_EmptyList — возвращает пустой список без ошибок
func TestHandleGetAds_EmptyList(t *testing.T) {
	h := newTestAdsHandlers(&mockAdsService{
		GetAllFunc: func() []models.Ad { return []models.Ad{} },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ads", nil)
	w := httptest.NewRecorder()

	h.HandleGetAds(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// TestHandleGetAds_MethodNotAllowed — POST-запрос возвращает 400
func TestHandleGetAds_MethodNotAllowed(t *testing.T) {
	h := newTestAdsHandlers(&mockAdsService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ads", nil)
	w := httptest.NewRecorder()

	h.HandleGetAds(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

// TestHandleGetAds_MethodPut — PUT-запрос также не разрешён → 400
func TestHandleGetAds_MethodPut(t *testing.T) {
	h := newTestAdsHandlers(&mockAdsService{})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/ads", nil)
	w := httptest.NewRecorder()

	h.HandleGetAds(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}
