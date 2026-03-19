package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// HandleLogout обрабатывает запросы на выход из системы, добавляя jti токена в черный список.
// @Summary Выход пользователя
// @Description Инвалидирует текущую сессию и очищает аутентификационную куку
// @Tags auth
// @Success 200 {object} map[string]string "logout successful"
// @Failure 401 {object} ErrorResponse "invalid or expired token"
// @Failure 500 {object} ErrorResponse "internal server error"
// @Router /auth/logout [post]
// @Security CookieAuth
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	h.log.Info("logout attempt", slog.String("op", "HandleLogout"))

	// достаём jti из контекста
	jti, ok := r.Context().Value(middleware.JtiKey).(string)
	if !ok {
		h.log.Error("jti not found in context")
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	if err := h.services.Auth.Logout(r.Context(), jti, time.Now().Add(h.tokenTTL)); err != nil {
		h.log.Error("failed to logout in service", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogout)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: "",
		// Domain: "clover-go.ru", // Убран хардкод домена
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,              // удаляем куку
		Expires:  time.Unix(0, 0), // на всякий случай делаем просроченной
	})

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
