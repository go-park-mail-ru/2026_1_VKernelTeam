// Package middleware реализует HTTP-middleware для Support-сервиса.
//
// В отличие от монолита, где JWT парсится локально,
// здесь токен валидируется через gRPC вызов к Auth-сервису.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// TokenValidator описывает валидацию токена через Auth gRPC.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID int64, role string, err error)
}

// GRPCAuthMiddleware проверяет JWT через Auth gRPC и кладёт userID в контекст.
// Использует те же ключи контекста (middleware.UserIDKey), что и монолит,
// чтобы хендлеры работали без изменений.
func GRPCAuthMiddleware(log *slog.Logger, validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				responser.RespondWithError(w, http.StatusUnauthorized, "missing token cookie")
				return
			}

			userID, _, err := validator.ValidateToken(r.Context(), cookie.Value)
			if err != nil {
				log.WarnContext(r.Context(), "gRPC token validation failed",
					slog.String("error", err.Error()),
				)
				responser.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), sharedmw.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
