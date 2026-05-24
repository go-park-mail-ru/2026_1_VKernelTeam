// Package review — usecase отзывов и рейтинга продавцов.
package review

//go:generate mockgen -source=review.go -destination=mocks/mock_review.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	reviewrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/review"
)

const (
	opCreateReview   = "usecase.review.CreateReview"
	opUpdateReview   = "usecase.review.UpdateReview"
	opDeleteReview   = "usecase.review.DeleteReview"
	opGetUserReviews = "usecase.review.GetUserReviews"
	opGetMyReviews   = "usecase.review.GetMyReviews"
	opGetSummary     = "usecase.review.GetSummary"

	defaultListLim = 20
	maxListLim     = 100
	minContentLen  = 5
	maxContentLen  = 2000
	minRating      = 1
	maxRating      = 5
)

// Бизнес-ошибки отзывов.
var (
	ErrNotPurchased        = errors.New("product was not purchased")
	ErrSellerMismatch      = errors.New("receiver is not the seller of this product")
	ErrSelfReview          = errors.New("cannot review yourself")
	ErrForbiddenReviewEdit = errors.New("not review author")
	ErrInvalidRating       = errors.New("rating must be between 1 and 5")
	ErrInvalidContent      = errors.New("content length must be between 5 and 2000")
)

// ReviewProvider — публичный контракт сервиса отзывов для handlers.
type ReviewProvider interface {
	CreateReview(ctx context.Context, senderID int64, req dto.CreateReviewRequest) (dto.ReviewResponse, error)
	UpdateReview(ctx context.Context, userID, reviewID int64, req dto.UpdateReviewRequest) (dto.ReviewResponse, error)
	DeleteReview(ctx context.Context, userID, reviewID int64) error
	GetUserReviews(ctx context.Context, receiverID int64, cursor *int64, limit int) (dto.ReviewListResponse, error)
	GetUserReviewsSummary(ctx context.Context, receiverID int64) (dto.ReviewSummaryResponse, error)
	GetMyReviews(ctx context.Context, senderID int64, cursor *int64, limit int) (dto.ReviewListResponse, error)
}

// ReviewStorage — узкий контракт хранилища, нужный сервису отзывов.
type ReviewStorage interface {
	Create(ctx context.Context, r *models.Review) (models.Review, error)
	Update(ctx context.Context, id, senderID int64, rating int, content string) (models.Review, error)
	Delete(ctx context.Context, id, senderID int64) error
	GetByID(ctx context.Context, id int64) (models.Review, error)
	GetResponseByID(ctx context.Context, reviewID int64) (dto.ReviewResponse, error)
	ListByReceiver(ctx context.Context, receiverID int64, cursor *int64, limit int) ([]dto.ReviewResponse, error)
	ListBySender(ctx context.Context, senderID int64, cursor *int64, limit int) ([]dto.ReviewResponse, error)
	SummaryByReceiver(ctx context.Context, receiverID int64) (dto.ReviewSummaryResponse, error)
	ExistsPurchaseRequest(ctx context.Context, buyerID, sellerID, productID int64) (bool, error)
	GetProductSellerID(ctx context.Context, productID int64) (int64, error)
}

// ReviewService реализует ReviewProvider.
type ReviewService struct {
	storage ReviewStorage
	log     *slog.Logger
}

// NewService создаёт сервис отзывов.
func NewService(storage ReviewStorage, log *slog.Logger) *ReviewService {
	return &ReviewService{storage: storage, log: log}
}

// CreateReview валидирует вход, проверяет факт покупки и продавца, создаёт отзыв.
// Триггер БД пересчитывает user.rating и user.reviews_count для receiver.
func (s *ReviewService) CreateReview(
	ctx context.Context, senderID int64, req dto.CreateReviewRequest,
) (dto.ReviewResponse, error) {
	if err := validateRating(req.Rating); err != nil {
		return dto.ReviewResponse{}, err
	}
	if err := validateContent(req.Content); err != nil {
		return dto.ReviewResponse{}, err
	}
	if senderID == req.ReceiverID {
		return dto.ReviewResponse{}, ErrSelfReview
	}

	sellerID, err := s.storage.GetProductSellerID(ctx, req.ProductID)
	if err != nil {
		if errors.Is(err, reviewrepo.ErrProductNotFound) {
			return dto.ReviewResponse{}, err
		}
		return dto.ReviewResponse{}, fmt.Errorf("CreateReview: get seller: %w", err)
	}
	if sellerID != req.ReceiverID {
		return dto.ReviewResponse{}, ErrSellerMismatch
	}

	purchased, err := s.storage.ExistsPurchaseRequest(ctx, senderID, req.ReceiverID, req.ProductID)
	if err != nil {
		return dto.ReviewResponse{}, fmt.Errorf("CreateReview: check purchase: %w", err)
	}
	if !purchased {
		return dto.ReviewResponse{}, ErrNotPurchased
	}

	created, err := s.storage.Create(ctx, &models.Review{
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		ProductID:  req.ProductID,
		Rating:     req.Rating,
		Content:    req.Content,
	})
	if err != nil {
		if errors.Is(err, reviewrepo.ErrReviewAlreadyExists) {
			return dto.ReviewResponse{}, err
		}
		s.log.ErrorContext(ctx, "failed to create review",
			slog.String("op", opCreateReview),
			slog.String("error", err.Error()),
			slog.Int64("sender_id", senderID),
			slog.Int64("receiver_id", req.ReceiverID),
			slog.Int64("product_id", req.ProductID),
		)
		return dto.ReviewResponse{}, fmt.Errorf("CreateReview: %w", err)
	}

	s.log.InfoContext(ctx, "review created",
		slog.String("op", opCreateReview),
		slog.Int64("review_id", created.ID),
		slog.Int64("sender_id", senderID),
		slog.Int64("receiver_id", req.ReceiverID),
		slog.Int64("product_id", req.ProductID),
	)

	resp, err := s.storage.GetResponseByID(ctx, created.ID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to fetch created review response",
			slog.String("op", opCreateReview),
			slog.String("error", err.Error()),
			slog.Int64("review_id", created.ID),
		)
		return dto.ReviewResponse{}, fmt.Errorf("CreateReview: fetch response: %w", err)
	}
	return resp, nil
}

// UpdateReview меняет rating/content отзыва, если userID — автор.
func (s *ReviewService) UpdateReview(
	ctx context.Context, userID, reviewID int64, req dto.UpdateReviewRequest,
) (dto.ReviewResponse, error) {
	existing, err := s.storage.GetByID(ctx, reviewID)
	if err != nil {
		if errors.Is(err, reviewrepo.ErrReviewNotFound) {
			return dto.ReviewResponse{}, err
		}
		return dto.ReviewResponse{}, fmt.Errorf("UpdateReview: get: %w", err)
	}
	if existing.SenderID != userID {
		return dto.ReviewResponse{}, ErrForbiddenReviewEdit
	}
	if err := validateRating(req.Rating); err != nil {
		return dto.ReviewResponse{}, err
	}
	if err := validateContent(req.Content); err != nil {
		return dto.ReviewResponse{}, err
	}

	updated, err := s.storage.Update(ctx, reviewID, userID, req.Rating, req.Content)
	if err != nil {
		if errors.Is(err, reviewrepo.ErrReviewNotFound) {
			return dto.ReviewResponse{}, err
		}
		s.log.ErrorContext(ctx, "failed to update review",
			slog.String("op", opUpdateReview),
			slog.String("error", err.Error()),
			slog.Int64("review_id", reviewID),
		)
		return dto.ReviewResponse{}, fmt.Errorf("UpdateReview: %w", err)
	}

	resp, err := s.storage.GetResponseByID(ctx, updated.ID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to fetch updated review response",
			slog.String("op", opUpdateReview),
			slog.String("error", err.Error()),
			slog.Int64("review_id", updated.ID),
		)
		return dto.ReviewResponse{}, fmt.Errorf("UpdateReview: fetch response: %w", err)
	}
	return resp, nil
}

// DeleteReview удаляет отзыв, если userID — его автор.
func (s *ReviewService) DeleteReview(ctx context.Context, userID, reviewID int64) error {
	existing, err := s.storage.GetByID(ctx, reviewID)
	if err != nil {
		if errors.Is(err, reviewrepo.ErrReviewNotFound) {
			return err
		}
		return fmt.Errorf("DeleteReview: get: %w", err)
	}
	if existing.SenderID != userID {
		return ErrForbiddenReviewEdit
	}
	if err := s.storage.Delete(ctx, reviewID, userID); err != nil {
		if errors.Is(err, reviewrepo.ErrReviewNotFound) {
			return err
		}
		s.log.ErrorContext(ctx, "failed to delete review",
			slog.String("op", opDeleteReview),
			slog.String("error", err.Error()),
			slog.Int64("review_id", reviewID),
		)
		return fmt.Errorf("DeleteReview: %w", err)
	}
	return nil
}

// GetUserReviews — страница отзывов о receiverID. Реализует look-ahead-курсор:
// запрашиваем limit+1, отрезаем хвост, в NextCursor кладём id последнего.
func (s *ReviewService) GetUserReviews(
	ctx context.Context, receiverID int64, cursor *int64, limit int,
) (dto.ReviewListResponse, error) {
	limit = clampLimit(limit)
	items, err := s.storage.ListByReceiver(ctx, receiverID, cursor, limit+1)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to list user reviews",
			slog.String("op", opGetUserReviews),
			slog.String("error", err.Error()),
			slog.Int64("receiver_id", receiverID),
		)
		return dto.ReviewListResponse{}, fmt.Errorf("GetUserReviews: %w", err)
	}
	return paginate(items, limit), nil
}

// GetMyReviews — страница отзывов, оставленных senderID.
func (s *ReviewService) GetMyReviews(
	ctx context.Context, senderID int64, cursor *int64, limit int,
) (dto.ReviewListResponse, error) {
	limit = clampLimit(limit)
	items, err := s.storage.ListBySender(ctx, senderID, cursor, limit+1)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to list my reviews",
			slog.String("op", opGetMyReviews),
			slog.String("error", err.Error()),
			slog.Int64("sender_id", senderID),
		)
		return dto.ReviewListResponse{}, fmt.Errorf("GetMyReviews: %w", err)
	}
	return paginate(items, limit), nil
}

// GetUserReviewsSummary возвращает агрегат отзывов о продавце.
func (s *ReviewService) GetUserReviewsSummary(
	ctx context.Context, receiverID int64,
) (dto.ReviewSummaryResponse, error) {
	resp, err := s.storage.SummaryByReceiver(ctx, receiverID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get reviews summary",
			slog.String("op", opGetSummary),
			slog.String("error", err.Error()),
			slog.Int64("receiver_id", receiverID),
		)
		return dto.ReviewSummaryResponse{}, fmt.Errorf("GetUserReviewsSummary: %w", err)
	}
	if resp.Distribution == nil {
		resp.Distribution = map[int]int{}
	}
	return resp, nil
}

func validateRating(rating int) error {
	if rating < minRating || rating > maxRating {
		return ErrInvalidRating
	}
	return nil
}

func validateContent(content string) error {
	l := len(content)
	if l < minContentLen || l > maxContentLen {
		return ErrInvalidContent
	}
	return nil
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLim
	}
	if limit > maxListLim {
		return maxListLim
	}
	return limit
}

// paginate отрезает хвост в limit+1 элементов и формирует курсор следующей страницы.
func paginate(items []dto.ReviewResponse, limit int) dto.ReviewListResponse {
	resp := dto.ReviewListResponse{Reviews: []dto.ReviewResponse{}}
	if len(items) > limit {
		last := items[limit-1].ID
		resp.Reviews = items[:limit]
		resp.NextCursor = &last
		return resp
	}
	resp.Reviews = items
	return resp
}
