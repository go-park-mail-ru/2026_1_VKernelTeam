package handlers

//go:generate mockgen -source=reviews.go -destination=mocks/mock_review_provider.go -package=mocks

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
	reviewrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/review"
	reviewuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/review"
)

const (
	opHandleCreateReview       = "handlers.HandleCreateReview"
	opHandleUpdateReview       = "handlers.HandleUpdateReview"
	opHandleDeleteReview       = "handlers.HandleDeleteReview"
	opHandleListUserReviews    = "handlers.HandleListUserReviews"
	opHandleUserReviewsSummary = "handlers.HandleUserReviewsSummary"
	opHandleListMyReviews      = "handlers.HandleListMyReviews"
)

// Текстовые ошибки клиента, попадающие в body как `{"error":"..."}`.
const (
	ErrInvalidReviewID  = "invalid review id"
	ErrInvalidUserID    = "invalid user id"
	ErrRatingRange      = "rating must be between 1 and 5"
	ErrContentRange     = "content length must be between 5 and 2000"
	ErrCannotReviewSelf = "cannot review yourself"
	ErrNotPurchased     = "product was not purchased"
	ErrSellerMismatch   = "receiver is not the seller of this product"
	ErrReviewExists     = "review already exists"
	ErrNotReviewAuthor  = "not review author"
	ErrReviewNotFound   = "review not found"
	ErrProductNotFound  = "product not found"
)

// ReviewProvider — usecase отзывов, нужный handlers.
type ReviewProvider interface {
	CreateReview(ctx context.Context, senderID int64, req dto.CreateReviewRequest) (dto.ReviewResponse, error)
	UpdateReview(ctx context.Context, userID, reviewID int64, req dto.UpdateReviewRequest) (dto.ReviewResponse, error)
	DeleteReview(ctx context.Context, userID, reviewID int64) error
	GetUserReviews(ctx context.Context, receiverID int64, cursor *int64, limit int) (dto.ReviewListResponse, error)
	GetUserReviewsSummary(ctx context.Context, receiverID int64) (dto.ReviewSummaryResponse, error)
	GetMyReviews(ctx context.Context, senderID int64, cursor *int64, limit int) (dto.ReviewListResponse, error)
}

// ReviewHandlers содержит обработчики отзывов.
type ReviewHandlers struct {
	log     *slog.Logger
	reviews ReviewProvider
}

// NewReviewHandlers создаёт обработчики отзывов.
func NewReviewHandlers(log *slog.Logger, reviews ReviewProvider) *ReviewHandlers {
	return &ReviewHandlers{log: log, reviews: reviews}
}

// HandleCreateReview создаёт отзыв.
// @Summary Создать отзыв о продавце
// @Tags reviews
// @Accept json
// @Produce json
// @Param body body dto.CreateReviewRequest true "receiver_id, product_id, rating, content"
// @Success 201 {object} dto.CreateReviewResponse
// @Failure 400 {object} dto.ErrorResponse "validation / business error"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /reviews [post]
func (h *ReviewHandlers) HandleCreateReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var req dto.CreateReviewRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.reviews.CreateReview(r.Context(), userID, req)
	if err != nil {
		h.respondReviewClientError(w, r, opHandleCreateReview, err)
		return
	}
	responser.RespondWithJSON(w, http.StatusCreated, dto.CreateReviewResponse{Review: resp})
}

// HandleUpdateReview обновляет свой отзыв.
// @Summary Обновить свой отзыв
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path int true "ID отзыва"
// @Param body body dto.UpdateReviewRequest true "rating, content"
// @Success 200 {object} dto.ReviewResponse
// @Failure 400 {object} dto.ErrorResponse "validation / not author / not found"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /reviews/{id} [put]
func (h *ReviewHandlers) HandleUpdateReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidReviewID)
		return
	}

	var req dto.UpdateReviewRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.reviews.UpdateReview(r.Context(), userID, reviewID, req)
	if err != nil {
		h.respondReviewClientError(w, r, opHandleUpdateReview, err)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleDeleteReview удаляет свой отзыв.
// @Summary Удалить свой отзыв
// @Tags reviews
// @Param id path int true "ID отзыва"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse "invalid id / not author / not found"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /reviews/{id} [delete]
func (h *ReviewHandlers) HandleDeleteReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidReviewID)
		return
	}

	if err := h.reviews.DeleteReview(r.Context(), userID, reviewID); err != nil {
		h.respondReviewClientError(w, r, opHandleDeleteReview, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleListUserReviews возвращает страницу отзывов о продавце.
// @Summary Отзывы о пользователе
// @Tags reviews
// @Produce json
// @Param id     path  int false "ID продавца"
// @Param cursor query int false "ID последнего элемента предыдущей страницы"
// @Param limit  query int false "лимит (по умолчанию 20, максимум 100)"
// @Success 200 {object} dto.ReviewListResponse
// @Failure 400 {object} dto.ErrorResponse "invalid user id"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /users/{id}/reviews [get]
func (h *ReviewHandlers) HandleListUserReviews(w http.ResponseWriter, r *http.Request) {
	receiverID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || receiverID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	cursor := parseCursor(r.URL.Query().Get("cursor"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	resp, err := h.reviews.GetUserReviews(r.Context(), receiverID, cursor, limit)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list user reviews",
			slog.String("op", opHandleListUserReviews),
			slog.Int64("receiver_id", receiverID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleUserReviewsSummary возвращает агрегат рейтинга продавца.
// @Summary Сводка отзывов о пользователе
// @Tags reviews
// @Produce json
// @Param id path int false "ID продавца"
// @Success 200 {object} dto.ReviewSummaryResponse
// @Failure 400 {object} dto.ErrorResponse "invalid user id"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /users/{id}/reviews/summary [get]
func (h *ReviewHandlers) HandleUserReviewsSummary(w http.ResponseWriter, r *http.Request) {
	receiverID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || receiverID <= 0 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	resp, err := h.reviews.GetUserReviewsSummary(r.Context(), receiverID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get reviews summary",
			slog.String("op", opHandleUserReviewsSummary),
			slog.Int64("receiver_id", receiverID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleListMyReviews возвращает отзывы, оставленные текущим пользователем.
// @Summary Мои отзывы
// @Tags reviews
// @Produce json
// @Param cursor query int false "ID последнего элемента предыдущей страницы"
// @Param limit  query int false "лимит (по умолчанию 20, максимум 100)"
// @Success 200 {object} dto.ReviewListResponse
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /profile/reviews [get]
func (h *ReviewHandlers) HandleListMyReviews(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	cursor := parseCursor(r.URL.Query().Get("cursor"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	resp, err := h.reviews.GetMyReviews(r.Context(), userID, cursor, limit)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to list my reviews",
			slog.String("op", opHandleListMyReviews),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// respondReviewClientError маппит usecase/repo-ошибки на 400 с текстом по дизайну.
// Все клиентские ошибки идут единым кодом 400, чтобы фронт не различал 403/404/409.
func (h *ReviewHandlers) respondReviewClientError(
	w http.ResponseWriter, r *http.Request, op string, err error,
) {
	switch {
	case errors.Is(err, reviewuc.ErrInvalidRating):
		responser.RespondWithError(w, http.StatusBadRequest, ErrRatingRange)
	case errors.Is(err, reviewuc.ErrInvalidContent):
		responser.RespondWithError(w, http.StatusBadRequest, ErrContentRange)
	case errors.Is(err, reviewuc.ErrSelfReview):
		responser.RespondWithError(w, http.StatusBadRequest, ErrCannotReviewSelf)
	case errors.Is(err, reviewuc.ErrNotPurchased):
		responser.RespondWithError(w, http.StatusBadRequest, ErrNotPurchased)
	case errors.Is(err, reviewuc.ErrSellerMismatch):
		responser.RespondWithError(w, http.StatusBadRequest, ErrSellerMismatch)
	case errors.Is(err, reviewuc.ErrForbiddenReviewEdit):
		responser.RespondWithError(w, http.StatusBadRequest, ErrNotReviewAuthor)
	case errors.Is(err, reviewrepo.ErrReviewAlreadyExists):
		responser.RespondWithError(w, http.StatusBadRequest, ErrReviewExists)
	case errors.Is(err, reviewrepo.ErrReviewNotFound):
		responser.RespondWithError(w, http.StatusBadRequest, ErrReviewNotFound)
	case errors.Is(err, reviewrepo.ErrProductNotFound):
		responser.RespondWithError(w, http.StatusBadRequest, ErrProductNotFound)
	default:
		h.log.ErrorContext(r.Context(), "review handler internal error",
			slog.String("op", op),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
	}
}

// parseCursor конвертирует строковый query-параметр в *int64. Пустая строка или
// невалидное значение трактуются как «без курсора».
func parseCursor(raw string) *int64 {
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
