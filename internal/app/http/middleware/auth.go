package middleware

import (
	"context"
	"log/slog"
	"net/http"

	utils "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage/blacklist"
	"github.com/golang-jwt/jwt/v5"
)

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
func AuthMiddleware(log *slog.Logger, bl *blacklist.InMemory, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, ErrMissingTokenCookie)
				return
			}

			tokenString := cookie.Value

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				log.Info("invalid token attempt", slog.String("error", err.Error()))
				utils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Error("failed to cast claims", slog.Any("claims", token.Claims))
				utils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidTokenClaims)
				return
			}

			// Блокируем запрос, если токен был отозван
			jti, _ := claims["jti"].(string)
			if bl.Check(jti) {
				log.Info("attempt to use revoked token", slog.String("jti", jti))
				utils.RespondWithError(w, http.StatusUnauthorized, ErrTokenRevoked)
				return
			}

			uidRaw, ok := claims["uid"].(float64)
			if !ok {
				utils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidUidClaim)
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
