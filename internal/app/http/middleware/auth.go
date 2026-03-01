package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware првоеряет каждый запрос.
func AuthMiddleware(bl *blacklist.InMemory, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.RespondWithError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid auth header format")
				return
			}

			tokenString := headerParts[1]

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
