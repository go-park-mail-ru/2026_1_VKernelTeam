package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubValidator struct {
	fn func(ctx context.Context, token string) (int64, string, error)
}

func (s stubValidator) ValidateToken(ctx context.Context, token string) (int64, string, error) {
	return s.fn(ctx, token)
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGRPCAuthMiddleware_NoCookie(t *testing.T) {
	mw := GRPCAuthMiddleware(newLogger(), stubValidator{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGRPCAuthMiddleware_InvalidToken(t *testing.T) {
	mw := GRPCAuthMiddleware(newLogger(), stubValidator{
		fn: func(ctx context.Context, token string) (int64, string, error) {
			return 0, "", errors.New("invalid")
		},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "bad"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGRPCAuthMiddleware_OK_PutsUserIDInContext(t *testing.T) {
	const wantUserID = int64(123)
	mw := GRPCAuthMiddleware(newLogger(), stubValidator{
		fn: func(ctx context.Context, token string) (int64, string, error) {
			return wantUserID, "user", nil
		},
	})

	var gotUserID int64
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(sharedmw.UserIDKey)
		uid, ok := v.(int64)
		require.True(t, ok, "expected int64 user id in ctx")
		gotUserID = uid
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "good"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, wantUserID, gotUserID)
}
