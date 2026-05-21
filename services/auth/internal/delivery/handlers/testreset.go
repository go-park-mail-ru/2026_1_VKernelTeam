package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/usecase/testreset"
)

const ErrOnlyTestUsers = "only test users can reset data"

// TestResetUseCase описывает поведение, нужное HTTP-слою.
type TestResetUseCase interface {
	Reset(ctx context.Context, userID int64) error
}

// TestResetHandlers — обработчики тестовой ручки очистки данных.
type TestResetHandlers struct {
	log *slog.Logger
	uc  TestResetUseCase
}

// NewTestResetHandlers создаёт обработчики тестовой очистки.
func NewTestResetHandlers(log *slog.Logger, uc TestResetUseCase) *TestResetHandlers {
	return &TestResetHandlers{log: log, uc: uc}
}

// HandleReset очищает связанные данные тестового пользователя.
// @Summary Очистить данные тестового пользователя
// @Description Удаляет объявления, корзину, избранное, чаты, отзывы, тикеты, кошелёк и сбрасывает аватар/рейтинг. Аккаунт остаётся. Доступно только email с префиксом "clover-tester".
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string "данные очищены"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 403 {object} dto.ErrorResponse "only test users can reset data"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Security CsrfHeaderAuth
// @Router /test/reset [post]
func (h *TestResetHandlers) HandleReset(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	if err := h.uc.Reset(r.Context(), userID); err != nil {
		if errors.Is(err, testreset.ErrNotTestUser) {
			responser.RespondWithError(w, http.StatusForbidden, ErrOnlyTestUsers)
			return
		}
		h.log.ErrorContext(r.Context(), "test reset failed",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
