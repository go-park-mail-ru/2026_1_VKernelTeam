package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/golang-jwt/jwt/v5"
)

// TokenChecker интерфейс для проверки отозванных токенов
type TokenChecker interface {
	Check(jti string) bool
}

// Свой тип для хранения ключей контекста
type contextKey string

// Ключи контекста, под которыми middleware кладёт данные аутентификации.
const (
	UserIDKey contextKey = "userID"
	JtiKey    contextKey = "jti"
)

// Сообщения об ошибках аутентификации, возвращаемые клиенту.
var (
	ErrMissingTokenCookie = "missing token cookie"
	ErrInvalidToken       = "invalid token"
	ErrInvalidTokenClaims = "invalid token claims"
	ErrTokenRevoked       = "token has been revoked"
	ErrInvalidUidClaim    = "invalid uid claim"
)

// AuthMiddleware проверяет каждый запрос.
func AuthMiddleware(log *slog.Logger, bl TokenChecker, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				responser.RespondWithError(w, http.StatusUnauthorized, ErrMissingTokenCookie)
				return
			}

			tokenString := cookie.Value

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				log.WarnContext(r.Context(), "invalid token attempt", slog.String("error", err.Error()))
				responser.RespondWithError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.ErrorContext(r.Context(), "failed to cast claims", slog.Any("claims", token.Claims))
				responser.RespondWithError(w, http.StatusUnauthorized, ErrInvalidTokenClaims)
				return
			}

			jti, _ := claims["jti"].(string)
			if bl.Check(jti) {
				log.WarnContext(r.Context(), "attempt to use revoked token", slog.String("jti", jti))
				responser.RespondWithError(w, http.StatusUnauthorized, ErrTokenRevoked)
				return
			}

			uidRaw, ok := claims["uid"].(float64)
			if !ok {
				responser.RespondWithError(w, http.StatusUnauthorized, ErrInvalidUidClaim)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, int64(uidRaw))
			ctx = context.WithValue(ctx, JtiKey, jti)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware пытается извлечь user_id из JWT cookie.
// Если токен отсутствует или невалиден — пропускает запрос без user_id в контексте.
func OptionalAuthMiddleware(log *slog.Logger, bl TokenChecker, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				log.DebugContext(r.Context(), "optional auth: invalid token, proceeding as anonymous")
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			jti, _ := claims["jti"].(string)
			if bl.Check(jti) {
				next.ServeHTTP(w, r)
				return
			}

			uidRaw, ok := claims["uid"].(float64)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, int64(uidRaw))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
