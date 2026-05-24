package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func withAdmin(r *http.Request, adminID int64) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, adminID)
	return r.WithContext(ctx)
}

func TestHandleAdminDeleteAd_Success(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().AdminDeleteAd(gomock.Any(), int64(7), int64(1)).Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/ads/{id}/admin", h.HandleAdminDeleteAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/ads/7/admin", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "deleted")
}

func TestHandleAdminDeleteAd_Unauthorized(t *testing.T) {
	h, _ := setupAdsHandlers(t)
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/ads/{id}/admin", h.HandleAdminDeleteAd)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/ads/7/admin", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleAdminDeleteAd_NotFound(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().AdminDeleteAd(gomock.Any(), int64(9), int64(1)).Return(ad.ErrAdNotFound)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/ads/{id}/admin", h.HandleAdminDeleteAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/ads/9/admin", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleAdminDeleteAd_InvalidID(t *testing.T) {
	h, _ := setupAdsHandlers(t)
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/ads/{id}/admin", h.HandleAdminDeleteAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/ads/abc/admin", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetModerationFlag(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().IsModerationEnabled(gomock.Any()).Return(true)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/admin/moderation/settings", nil)
	rr := httptest.NewRecorder()
	h.HandleGetModerationFlag(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]bool
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.True(t, resp["enabled"])
}

func TestHandleSetModerationFlag_OK(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().SetModerationEnabled(gomock.Any(), true, int64(1)).Return(nil)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/admin/moderation/settings",
		strings.NewReader(`{"enabled":true}`)), 1)
	rr := httptest.NewRecorder()
	h.HandleSetModerationFlag(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleSetModerationFlag_BadJSON(t *testing.T) {
	h, _ := setupAdsHandlers(t)
	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/admin/moderation/settings",
		strings.NewReader(`broken`)), 1)
	rr := httptest.NewRecorder()
	h.HandleSetModerationFlag(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetModerationQueue(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().GetModerationQueue(gomock.Any()).Return([]models.Ad{{ID: 1}}, nil)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/admin/moderation/queue", nil)
	rr := httptest.NewRecorder()
	h.HandleGetModerationQueue(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":1`)
}

func TestHandleApproveAd_Success(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().ApproveAd(gomock.Any(), int64(3), int64(1)).Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/admin/moderation/ads/{id}/approve", h.HandleApproveAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/admin/moderation/ads/3/approve", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleApproveAd_NotFound(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().ApproveAd(gomock.Any(), int64(3), int64(1)).Return(ad.ErrAdNotFound)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/admin/moderation/ads/{id}/approve", h.HandleApproveAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/admin/moderation/ads/3/approve", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleRejectAd_WithReason(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().RejectAd(gomock.Any(), int64(4), int64(1), "spam").Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/admin/moderation/ads/{id}/reject", h.HandleRejectAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/admin/moderation/ads/4/reject",
		strings.NewReader(`{"reason":"spam"}`)), 1)
	req.ContentLength = int64(len(`{"reason":"spam"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleRejectAd_NoBody(t *testing.T) {
	h, mockAds := setupAdsHandlers(t)
	mockAds.EXPECT().RejectAd(gomock.Any(), int64(4), int64(1), "").Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/admin/moderation/ads/{id}/reject", h.HandleRejectAd)

	req := withAdmin(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/admin/moderation/ads/4/reject", nil), 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
