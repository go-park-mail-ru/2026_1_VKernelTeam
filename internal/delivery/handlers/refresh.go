package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// HandleRefresh обновляет access токен по refresh токену
// @Summary Обновление токенов
// @Description Получает новый access и rottates refresh token
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string "tokens refreshed"
// @Failure 401 {object} ErrorResponse "refresh token required / invalid refresh token: Ошибка refresh токена"
// @Failure 500 {object} ErrorResponse "internal error: Ошибка сервера при обновлении токенов"
// @Router /auth/refresh [post]
func (h *AuthHandlers) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		h.log.Info("refresh token missing")
		responser.RespondWithError(w, http.StatusUnauthorized, "refresh token required")
		return
	}

	newAccess, newRefresh, err := h.services.Auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.log.Error("failed to refresh tokens", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()

	h.setAuthCookie(w, newAccess, csrfToken)
	h.setRefreshCookie(w, newRefresh)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
