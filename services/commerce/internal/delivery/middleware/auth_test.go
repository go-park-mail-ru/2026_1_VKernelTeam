package middleware

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/middleware/mocks"
	"github.com/golang/mock/gomock"
)

func newTestLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestGRPCAuthMiddleware_NoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	mw := GRPCAuthMiddleware(newTestLog(), mocks.NewMockTokenValidator(ctrl))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestGRPCAuthMiddleware_InvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	tv := mocks.NewMockTokenValidator(ctrl)
	tv.EXPECT().ValidateToken(gomock.Any(), "x").Return(int64(0), "", errors.New("bad token"))

	mw := GRPCAuthMiddleware(newTestLog(), tv)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "x"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestGRPCAuthMiddleware_PutsUserIDInContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	tv := mocks.NewMockTokenValidator(ctrl)
	tv.EXPECT().ValidateToken(gomock.Any(), "x").Return(int64(42), "user", nil)

	mw := GRPCAuthMiddleware(newTestLog(), tv)
	var got int64
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = r.Context().Value(sharedmw.UserIDKey).(int64)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "x"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 || got != 42 {
		t.Fatalf("want 200/42, got %d/%d", rr.Code, got)
	}
}
