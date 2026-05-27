package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// Сообщения об ошибках проверки роли, возвращаемые клиенту.
const (
	ErrRoleForbidden = "forbidden"
)

// RoleProvider описывает источник роли пользователя по его ID.
type RoleProvider interface {
	GetUserRole(ctx context.Context, userID int64) (string, error)
}

// RequireRole пропускает только запросы, в контексте которых RoleKey равен одному
// из allowedRoles. Не делает дополнительных RPC: предполагает, что RoleKey уже
// положен в контекст в auth-middleware.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(RoleKey).(string)
			if _, ok := allowed[role]; !ok {
				// 401 (а не 403) — для соответствия минимальному набору кодов проекта.
				responser.RespondWithError(w, http.StatusUnauthorized, ErrRoleForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RoleMiddleware пропускает только пользователей с ролью из allowedRoles.
// Используется после AuthMiddleware: ожидает userID в контексте под UserIDKey.
func RoleMiddleware(log *slog.Logger, provider RoleProvider, allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(int64)
			if !ok || userID == 0 {
				responser.RespondWithError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			role, err := provider.GetUserRole(r.Context(), userID)
			if err != nil {
				log.ErrorContext(r.Context(), "failed to get user role",
					slog.Int64("user_id", userID),
					slog.String("error", err.Error()),
				)
				responser.RespondWithError(w, http.StatusInternalServerError, ErrRoleForbidden)
				return
			}

			if _, ok := allowed[role]; !ok {
				log.WarnContext(r.Context(), "role check failed",
					slog.Int64("user_id", userID),
					slog.String("role", role),
				)
				responser.RespondWithError(w, http.StatusBadRequest, ErrRoleForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
