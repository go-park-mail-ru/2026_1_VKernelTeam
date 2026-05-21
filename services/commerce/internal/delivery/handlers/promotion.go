package handlers

//go:generate mockgen -source=promotion.go -destination=mocks/mock_promotion_handlers.go -package=mocks

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
	commercemetrics "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/metrics"
	promouc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/promotion"
)

const (
	opHandleGetPlans       = "handlers.HandleGetPromotionPlans"
	opHandlePurchasePromo  = "handlers.HandlePurchasePromotion"
	opHandleListAdPromos   = "handlers.HandleListAdPromotions"
	opHandleListUserPromos = "handlers.HandleListUserPromotions"
)

// PromotionService — usecase продвижения, нужный хендлерам.
type PromotionService interface {
	GetPlans(ctx context.Context) ([]dto.PromotionPlanResponse, error)
	Purchase(ctx context.Context, userID, productID int64, planCode, idempotencyKey string) (dto.PurchasePromotionResponse, error)
	ListActiveByAd(ctx context.Context, productID int64) ([]dto.PromotionResponse, error)
	ListByUser(ctx context.Context, userID, cursor int64, limit int) (dto.PromotionListResponse, error)
}

// PromotionHandlers содержит обработчики промо.
type PromotionHandlers struct {
	log       *slog.Logger
	promotion PromotionService
}

func NewPromotionHandlers(log *slog.Logger, promotion PromotionService) *PromotionHandlers {
	return &PromotionHandlers{log: log, promotion: promotion}
}

// HandleGetPromotionPlans возвращает каталог активных тарифов.
// @Summary Список тарифов продвижения
// @Tags promotion
// @Produce json
// @Success 200 {array} dto.PromotionPlanResponse
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /promotion/plans [get]
func (h *PromotionHandlers) HandleGetPromotionPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.promotion.GetPlans(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get promotion plans",
			slog.String("op", opHandleGetPlans),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, plans)
}

// HandlePurchasePromotion покупает промо для объявления.
// @Summary Купить продвижение объявления
// @Tags promotion
// @Accept json
// @Produce json
// @Param id   path int true "ID объявления"
// @Param body body dto.PurchasePromotionRequest true "plan_code и idempotency_key"
// @Success 200 {object} dto.PurchasePromotionResponse
// @Failure 400 {object} dto.ErrorResponse "INSUFFICIENT_FUNDS / PLAN_NOT_FOUND / PLAN_INACTIVE / INVALID_AD_STATUS / INVALID_REQUEST"
// @Failure 403 {object} dto.ErrorResponse "NOT_AD_OWNER"
// @Failure 404 {object} dto.ErrorResponse "AD_NOT_FOUND"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /ads/{id}/promotions [post]
func (h *PromotionHandlers) HandlePurchasePromotion(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	var req dto.PurchasePromotionRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.promotion.Purchase(r.Context(), userID, productID, req.PlanCode, req.IdempotencyKey)
	if err != nil {
		h.respondPurchaseError(w, r, userID, productID, err)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *PromotionHandlers) respondPurchaseError(
	w http.ResponseWriter, r *http.Request, userID, productID int64, err error,
) {
	errorCounter := commercemetrics.Get().PromotionPurchaseErrors
	switch {
	case errors.Is(err, promouc.ErrAdNotFound):
		errorCounter.WithLabelValues("AD_NOT_FOUND").Inc()
		responser.RespondWithError(w, http.StatusNotFound, "AD_NOT_FOUND")
	case errors.Is(err, promouc.ErrPlanNotFound):
		errorCounter.WithLabelValues("PLAN_NOT_FOUND").Inc()
		responser.RespondWithError(w, http.StatusBadRequest, "PLAN_NOT_FOUND")
	case errors.Is(err, promouc.ErrPlanInactive):
		errorCounter.WithLabelValues("PLAN_INACTIVE").Inc()
		responser.RespondWithError(w, http.StatusBadRequest, "PLAN_INACTIVE")
	case errors.Is(err, promouc.ErrNotAdOwner):
		errorCounter.WithLabelValues("NOT_AD_OWNER").Inc()
		responser.RespondWithError(w, http.StatusForbidden, "NOT_AD_OWNER")
	case errors.Is(err, promouc.ErrInvalidAdStatus):
		errorCounter.WithLabelValues("INVALID_AD_STATUS").Inc()
		responser.RespondWithError(w, http.StatusBadRequest, "INVALID_AD_STATUS")
	case errors.Is(err, promouc.ErrInsufficientFund):
		errorCounter.WithLabelValues("INSUFFICIENT_FUNDS").Inc()
		responser.RespondWithError(w, http.StatusBadRequest, "INSUFFICIENT_FUNDS")
	case errors.Is(err, promouc.ErrInvalidRequest):
		errorCounter.WithLabelValues("INVALID_REQUEST").Inc()
		responser.RespondWithError(w, http.StatusBadRequest, "INVALID_REQUEST")
	default:
		errorCounter.WithLabelValues("INTERNAL").Inc()
		h.log.ErrorContext(r.Context(), "failed to purchase promotion",
			slog.String("op", opHandlePurchasePromo),
			slog.Int64("user_id", userID),
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
	}
}

// HandleListAdPromotions возвращает активные промо по объявлению.
// @Summary Активные промо объявления
// @Tags promotion
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {array} dto.PromotionResponse
// @Failure 400 {object} dto.ErrorResponse "invalid ad id"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /ads/{id}/promotions [get]
func (h *PromotionHandlers) HandleListAdPromotions(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	items, err := h.promotion.ListActiveByAd(r.Context(), productID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list ad promotions",
			slog.String("op", opHandleListAdPromos),
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, items)
}

// HandleListUserPromotions возвращает историю промо пользователя.
// @Summary История покупок продвижения
// @Tags promotion
// @Produce json
// @Param limit  query int false "лимит (по умолчанию 20, максимум 100)"
// @Param cursor query int false "ID последнего элемента предыдущей страницы"
// @Success 200 {object} dto.PromotionListResponse
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /profile/promotions [get]
func (h *PromotionHandlers) HandleListUserPromotions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)

	resp, err := h.promotion.ListByUser(r.Context(), userID, cursor, limit)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list user promotions",
			slog.String("op", opHandleListUserPromos),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}
