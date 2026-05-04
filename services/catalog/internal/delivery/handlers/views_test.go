package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/handlers/mocks"
	"github.com/golang/mock/gomock"
)

func newViewsMock(t *testing.T) (*ViewsHandlers, *mocks.MockViews) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mv := mocks.NewMockViews(ctrl)
	h := NewViewsHandlers(slog.New(slog.NewTextHandler(io.Discard, nil)), mv)
	return h, mv
}

func TestHandleRecordView_InvalidAdID(t *testing.T) {
	h, _ := newViewsMock(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ads/abc/view", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	h.HandleRecordView(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestHandleRecordView_MissingDeviceID(t *testing.T) {
	h, _ := newViewsMock(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ads/1/view", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.HandleRecordView(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestHandleRecordView_Success(t *testing.T) {
	h, mv := newViewsMock(t)
	mv.EXPECT().RecordView(
		gomock.Any(), int64(1), gomock.Nil(), "550e8400-e29b-41d4-a716-446655440000",
	).Return(int64(10), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ads/1/view", nil)
	req.SetPathValue("id", "1")
	req.Header.Set("X-Device-ID", "550e8400-e29b-41d4-a716-446655440000")
	rr := httptest.NewRecorder()
	h.HandleRecordView(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRecordView_UsecaseError(t *testing.T) {
	h, mv := newViewsMock(t)
	mv.EXPECT().RecordView(gomock.Any(), int64(1), gomock.Nil(), gomock.Any()).
		Return(int64(0), errors.New("redis down"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ads/1/view", nil)
	req.SetPathValue("id", "1")
	req.Header.Set("X-Device-ID", "550e8400-e29b-41d4-a716-446655440000")
	rr := httptest.NewRecorder()
	h.HandleRecordView(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
}
