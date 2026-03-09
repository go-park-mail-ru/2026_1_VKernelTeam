package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	ssntjwt "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/lib/jwt"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage/blacklist"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	// инициализируем логгер, который ничего не выводит (Discard), чтобы не спамить в консоль тестов
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	secret := "test-secret"
	bl := blacklist.New(time.Minute)

	mw := AuthMiddleware(log, bl, secret)

	// заглушка следующего обработчика
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, okUID := r.Context().Value(UserIDKey).(int64)
		jti, okJTI := r.Context().Value(JtiKey).(string)

		assert.True(t, okUID, "userID should be in context")
		assert.Equal(t, int64(42), userID)

		assert.True(t, okJTI, "jti should be in context")
		assert.NotEmpty(t, jti)

		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := mw(nextHandler)

	user := models.User{ID: 42, Email: "test@example.com"}
	validToken, err := ssntjwt.NewToken(user, time.Hour, secret)
	require.NoError(t, err)

	t.Run("MissingToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "missing token cookie")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "invalid.token.str"})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid token")
	})

	t.Run("ValidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("RevokedToken", func(t *testing.T) {
		parsedToken, _ := jwt.Parse(validToken, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
		claims, _ := parsedToken.Claims.(jwt.MapClaims)
		jti := claims["jti"].(string)

		bl.Add(jti, time.Now().Add(time.Hour))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: validToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "token has been revoked")
	})

	t.Run("InvalidClaims", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
		invalidClaimsToken, _ := token.SignedString([]byte(secret))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: invalidClaimsToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
