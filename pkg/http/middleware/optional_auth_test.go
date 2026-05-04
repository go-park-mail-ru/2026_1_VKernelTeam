package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalAuthMiddleware(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := "test-secret"
	bl := &mockTokenChecker{revoked: make(map[string]bool)}
	mw := OptionalAuthMiddleware(log, bl, secret)

	makeHandler := func(t *testing.T, expectUserID bool, wantUID int64) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := r.Context().Value(UserIDKey).(int64)
			if expectUserID {
				assert.True(t, ok)
				assert.Equal(t, wantUID, uid)
			} else {
				assert.False(t, ok)
			}
			w.WriteHeader(http.StatusOK)
		})
	}

	t.Run("no cookie -> anonymous, 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(makeHandler(t, false, 0)).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid token -> anonymous, 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "garbage"})
		rec := httptest.NewRecorder()
		mw(makeHandler(t, false, 0)).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("valid token -> uid in ctx", func(t *testing.T) {
		tok := mintTestToken(t, 77, time.Hour, secret)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: tok})
		rec := httptest.NewRecorder()
		mw(makeHandler(t, true, 77)).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("revoked token -> anonymous, 200", func(t *testing.T) {
		tok := mintTestToken(t, 88, time.Hour, secret)
		parsed, _ := jwt.Parse(tok, func(*jwt.Token) (any, error) { return []byte(secret), nil })
		claims, _ := parsed.Claims.(jwt.MapClaims)
		jti, _ := claims["jti"].(string)
		require.NotEmpty(t, jti)
		bl.revoked[jti] = true

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: tok})
		rec := httptest.NewRecorder()
		mw(makeHandler(t, false, 0)).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("token without uid claim -> anonymous", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp": time.Now().Add(time.Hour).Unix(),
			// no uid
		}
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: tok})
		rec := httptest.NewRecorder()
		mw(makeHandler(t, false, 0)).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
