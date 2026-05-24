package handlers

//go:generate mockgen -source=purchase.go -destination=mocks/mock_purchase_provider.go -package=mocks

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

const (
	opHandleListMyPurchases = "handlers.HandleListMyPurchases"
)

// PurchaseProvider — usecase покупок, нужный handler-у.
type PurchaseProvider interface {
	GetMyPurchases(ctx context.Context, buyerID int64, cursor *int64, limit int) (dto.PurchaseListResponse, error)
}

// PurchaseHandlers содержит обработчики раздела «Мои покупки».
type PurchaseHandlers struct {
	log       *slog.Logger
	purchases PurchaseProvider
}

// NewPurchaseHandlers создаёт обработчики покупок.
func NewPurchaseHandlers(log *slog.Logger, purchases PurchaseProvider) *PurchaseHandlers {
	return &PurchaseHandlers{log: log, purchases: purchases}
}

// HandleListMyPurchases возвращает страницу покупок текущего пользователя.
// @Summary Мои покупки
// @Tags purchases
// @Produce json
// @Param cursor query int false "ID последнего элемента предыдущей страницы"
// @Param limit  query int false "лимит (по умолчанию 20, максимум 50)"
// @Success 200 {object} dto.PurchaseListResponse
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /profile/purchases [get]
func (h *PurchaseHandlers) HandleListMyPurchases(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	cursor := parseCursor(r.URL.Query().Get("cursor"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	resp, err := h.purchases.GetMyPurchases(r.Context(), userID, cursor, limit)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list my purchases",
			slog.String("op", opHandleListMyPurchases),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}
