package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cookieNameToken — имя cookie с JWT, используемое в тестах middleware.
const cookieNameToken = "token"

// mintTestToken — локальная мини-реализация подписи JWT для тестов middleware.
// Раньше использовался удалённый pkg/jwt из монолита; теперь auth-сервис
// держит свою копию у себя, а тесту middleware достаточно прямого вызова
// golang-jwt с теми же claims.
func mintTestToken(t *testing.T, userID int64, ttl time.Duration, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"uid": userID,
		"exp": time.Now().Add(ttl).Unix(),
		"jti": uuid.NewString(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return token
}

type mockTokenChecker struct {
	revoked map[string]bool
}

func (m *mockTokenChecker) Check(jti string) bool {
	return m.revoked[jti]
}

func (m *mockTokenChecker) Add(jti string, _ time.Time) {
	m.revoked[jti] = true
}

func TestAuthMiddleware(t *testing.T) {
	// инициализируем логгер, который ничего не выводит (Discard), чтобы не спамить в консоль тестов
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	secret := "test-secret"
	bl := &mockTokenChecker{revoked: make(map[string]bool)}

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

	validToken := mintTestToken(t, 42, time.Hour, secret)

	t.Run("MissingToken", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "missing token cookie")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: "invalid.token.str"})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid token")
	})

	t.Run("ValidToken", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: validToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("RevokedToken", func(t *testing.T) {
		parsedToken, _ := jwt.Parse(validToken, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
		claims, _ := parsedToken.Claims.(jwt.MapClaims)
		jti := claims["jti"].(string)

		bl.Add(jti, time.Now().Add(time.Hour))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: validToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "token has been revoked")
	})

	t.Run("InvalidClaims", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
		invalidClaimsToken, _ := token.SignedString([]byte(secret))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieNameToken, Value: invalidClaimsToken})
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
