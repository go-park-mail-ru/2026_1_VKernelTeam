package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware проверяет каждый запрос.
func AuthMiddleware(bl *blacklist.InMemory, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "missing token cookie")
				return
			}

			tokenString := cookie.Value

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// Блокируем запрос, если токен был отозван
			jti, _ := claims["jti"].(string)
			if bl.Check(jti) {
				utils.RespondWithError(w, http.StatusUnauthorized, "token has been revoked")
				return
			}

			uid := claims["uid"].(float64)
			ctx := context.WithValue(r.Context(), "userID", int64(uid))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
