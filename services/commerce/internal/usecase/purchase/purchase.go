// Package purchase — usecase покупок текущего пользователя.
package purchase

//go:generate mockgen -source=purchase.go -destination=mocks/mock_purchase.go -package=mocks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

const (
	opGetMyPurchases = "usecase.purchase.GetMyPurchases"
	defaultListLim   = 20
	maxListLim       = 50
)

// PurchaseRepo — выборка покупок из хранилища.
type PurchaseRepo interface {
	ListByBuyer(ctx context.Context, buyerID int64, cursor *int64, limit int) ([]dto.PurchaseItem, error)
}

// Service — usecase покупок.
type Service struct {
	log     *slog.Logger
	storage PurchaseRepo
}

// NewService создаёт usecase покупок.
func NewService(storage PurchaseRepo, log *slog.Logger) *Service {
	return &Service{log: log, storage: storage}
}

// GetMyPurchases — страница покупок buyerID. Look-ahead-курсор: запрашиваем
// limit+1 элементов, отрезаем хвост, в NextCursor кладём id последнего.
func (s *Service) GetMyPurchases(
	ctx context.Context, buyerID int64, cursor *int64, limit int,
) (dto.PurchaseListResponse, error) {
	limit = clampLimit(limit)
	items, err := s.storage.ListByBuyer(ctx, buyerID, cursor, limit+1)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to list my purchases",
			slog.String("op", opGetMyPurchases),
			slog.String("error", err.Error()),
			slog.Int64("buyer_id", buyerID),
		)
		return dto.PurchaseListResponse{}, fmt.Errorf("GetMyPurchases: %w", err)
	}
	return paginate(items, limit), nil
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

func paginate(items []dto.PurchaseItem, limit int) dto.PurchaseListResponse {
	resp := dto.PurchaseListResponse{Purchases: []dto.PurchaseItem{}}
	if len(items) > limit {
		last := items[limit-1].OrderID
		resp.Purchases = items[:limit]
		resp.NextCursor = &last
		return resp
	}
	resp.Purchases = items
	return resp
}
