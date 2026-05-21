package handlers

//go:generate mockgen -source=wallet.go -destination=mocks/mock_wallet_handlers.go -package=mocks

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mailru/easyjson"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	walletuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet"
)

const (
	opHandleGetWallet     = "handlers.HandleGetWallet"
	opHandleListWalletTxs = "handlers.HandleListWalletTransactions"
	opHandleTopupWallet   = "handlers.HandleTopupWallet"
)

// WalletService — usecase кошелька, нужный хендлерам.
type WalletService interface {
	GetBalance(ctx context.Context, userID int64) (dto.WalletResponse, error)
	ListTransactions(ctx context.Context, userID, cursor int64, limit int) (dto.WalletTransactionListResponse, error)
	Topup(ctx context.Context, userID, amount int64, idempotencyKey string) (dto.TopupWalletResponse, error)
}

// WalletHandlers содержит обработчики кошелька.
type WalletHandlers struct {
	log    *slog.Logger
	wallet WalletService
}

// NewWalletHandlers создаёт обработчики кошелька.
func NewWalletHandlers(log *slog.Logger, wallet WalletService) *WalletHandlers {
	return &WalletHandlers{log: log, wallet: wallet}
}

// HandleGetWallet возвращает текущий баланс пользователя.
// @Summary Получить баланс кошелька
// @Tags wallet
// @Produce json
// @Success 200 {object} dto.WalletResponse
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /wallet [get]
func (h *WalletHandlers) HandleGetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	resp, err := h.wallet.GetBalance(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get wallet",
			slog.String("op", opHandleGetWallet),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleListWalletTransactions возвращает ленту операций.
// @Summary История операций по кошельку
// @Tags wallet
// @Produce json
// @Param limit  query int false "лимит (по умолчанию 20, максимум 100)"
// @Param cursor query int false "ID последнего элемента предыдущей страницы"
// @Success 200 {object} dto.WalletTransactionListResponse
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /wallet/transactions [get]
func (h *WalletHandlers) HandleListWalletTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)

	resp, err := h.wallet.ListTransactions(r.Context(), userID, cursor, limit)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list wallet transactions",
			slog.String("op", opHandleListWalletTxs),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleTopupWallet пополняет кошелёк через мок-провайдера.
// @Summary Пополнить кошелёк
// @Tags wallet
// @Accept json
// @Produce json
// @Param body body dto.TopupWalletRequest true "Сумма в рублях и idempotency_key"
// @Success 200 {object} dto.TopupWalletResponse
// @Failure 400 {object} dto.ErrorResponse "INVALID_AMOUNT / INVALID_REQUEST"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /wallet/topup [post]
func (h *WalletHandlers) HandleTopupWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var req dto.TopupWalletRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.wallet.Topup(r.Context(), userID, req.Amount, req.IdempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, walletuc.ErrInvalidAmount), errors.Is(err, walletuc.ErrAmountTooLarge):
			responser.RespondWithError(w, http.StatusBadRequest, "INVALID_AMOUNT")
			return
		case errors.Is(err, walletuc.ErrInvalidRequest):
			responser.RespondWithError(w, http.StatusBadRequest, "INVALID_REQUEST")
			return
		}
		h.log.ErrorContext(r.Context(), "failed to topup wallet",
			slog.String("op", opHandleTopupWallet),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}
