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

const (
	UserIDKey contextKey = "userID"
	JtiKey    contextKey = "jti"
)

// ошибки middleware
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

			// Блокируем запрос, если токен был отозван
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

			// Кладём в контекст ID пользователя и jti
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, int64(uidRaw))
			ctx = context.WithValue(ctx, JtiKey, jti)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
