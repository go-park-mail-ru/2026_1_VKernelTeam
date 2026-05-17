package middleware

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/middleware/mocks"
	"github.com/golang/mock/gomock"
)

const tokenCookieName = "token"

func newTestLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newValidator(t *testing.T) *mocks.MockTokenValidator {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return mocks.NewMockTokenValidator(ctrl)
}

func TestGRPCAuthMiddleware_NoCookie(t *testing.T) {
	mw := GRPCAuthMiddleware(newTestLog(), newValidator(t))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestGRPCAuthMiddleware_InvalidToken(t *testing.T) {
	v := newValidator(t)
	v.EXPECT().ValidateToken(gomock.Any(), "x").Return(int64(0), "", errors.New("bad"))

	mw := GRPCAuthMiddleware(newTestLog(), v)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "x"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestGRPCAuthMiddleware_ValidPutsUserIDInContext(t *testing.T) {
	v := newValidator(t)
	v.EXPECT().ValidateToken(gomock.Any(), "x").Return(int64(42), "user", nil)

	mw := GRPCAuthMiddleware(newTestLog(), v)
	var got int64
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = r.Context().Value(sharedmw.UserIDKey).(int64)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "x"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 || got != 42 {
		t.Fatalf("want 200/42, got %d/%d", rr.Code, got)
	}
}

func TestGRPCOptionalAuthMiddleware_NoCookieStillCallsNext(t *testing.T) {
	mw := GRPCOptionalAuthMiddleware(newTestLog(), newValidator(t))
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if !called || rr.Code != 200 {
		t.Fatalf("expected next called and 200, got called=%v code=%d", called, rr.Code)
	}
}

func TestGRPCOptionalAuthMiddleware_ValidPutsUserIDInContext(t *testing.T) {
	v := newValidator(t)
	v.EXPECT().ValidateToken(gomock.Any(), "x").Return(int64(7), "user", nil)

	mw := GRPCOptionalAuthMiddleware(newTestLog(), v)
	var got int64
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = r.Context().Value(sharedmw.UserIDKey).(int64)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "x"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 || got != 7 {
		t.Fatalf("want 200/7, got %d/%d", rr.Code, got)
	}
}
