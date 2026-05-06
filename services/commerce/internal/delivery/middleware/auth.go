// Package middleware — HTTP middleware commerce-сервиса.
//
// Авторизация делегирована Auth gRPC; CSRF/CORS/RequestID/AccessLog берутся из pkg/http/middleware.
package middleware

//go:generate mockgen -source=auth.go -destination=mocks/mock_auth.go -package=mocks

import (
	"context"
	"log/slog"
	"net/http"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// TokenValidator — gRPC-валидация токена (реализуется AuthClient).
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID int64, role string, err error)
}

// GRPCAuthMiddleware кладёт user_id в контекст или возвращает 401.
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
				log.WarnContext(r.Context(), "gRPC token validation failed", slog.String("error", err.Error()))
				responser.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), sharedmw.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
