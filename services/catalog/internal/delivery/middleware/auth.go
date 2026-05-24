// Package middleware - HTTP-middleware catalog-сервиса.
//
// JWT валидируется через gRPC к Auth-сервису, чтобы catalog не знал про
// токен-секрет и blacklist (в монолите они были рядом, теперь живут только в Auth).
package middleware

//go:generate mockgen -source=auth.go -destination=mocks/mock_auth.go -package=mocks

import (
	"context"
	"log/slog"
	"net/http"

	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// TokenValidator - валидация токена через Auth gRPC.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID int64, role string, err error)
}

// GRPCAuthMiddleware - обязательная авторизация. Без cookie token -> 401.
func GRPCAuthMiddleware(log *slog.Logger, validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				responser.RespondWithError(w, http.StatusUnauthorized, "missing token cookie")
				return
			}

			userID, role, err := validator.ValidateToken(r.Context(), cookie.Value)
			if err != nil {
				log.WarnContext(r.Context(), "gRPC token validation failed",
					slog.String("error", err.Error()),
				)
				responser.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), sharedmw.UserIDKey, userID)
			ctx = context.WithValue(ctx, sharedmw.RoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GRPCOptionalAuthMiddleware - опциональная авторизация: если токен валиден,
// userID кладётся в контекст; если нет - запрос идёт дальше анонимно.
// Используется для /ads/{id}/view (просмотры пишутся и от незалогиненных).
func GRPCOptionalAuthMiddleware(log *slog.Logger, validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			userID, role, err := validator.ValidateToken(r.Context(), cookie.Value)
			if err != nil {
				log.DebugContext(r.Context(), "optional auth: invalid token, anonymous",
					slog.String("error", err.Error()),
				)
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), sharedmw.UserIDKey, userID)
			ctx = context.WithValue(ctx, sharedmw.RoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
