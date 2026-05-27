// Package grpc - реализация CatalogService gRPC API.
//
// Используется сервисом Commerce для:
//   - GetAd / GetAdsByIDs   - детали товаров (карточки в чатах, корзине)
//   - CheckAdStatus         - проверка перед добавлением в корзину
//   - UpdateAdStatus        - смена active -> reserved -> sold при покупке
//
// Аутентификация на gRPC сейчас не реализована (доверяем внутренней сети
// docker-compose). Для прод-инсталляции - mTLS или auth-interceptor.
package grpc

//go:generate mockgen -source=server.go -destination=mocks/mock_server.go -package=mocks

import (
	"context"
	"errors"
	"log/slog"

	catalogv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/catalog/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	adrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdProvider — методы, нужные gRPC-серверу. Реализуется adrepo.AdStorage.
type AdProvider interface {
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	UpdateAdStatus(ctx context.Context, id int64, newStatus string) (prevStatus string, err error)
}

// EventPublisher публикует события объявлений (для ad.sold при UpdateAdStatus).
// Может быть nil - события не публикуются.
type EventPublisher interface {
	PublishAdSold(ctx context.Context, adID, buyerID int64) error
}

// Server реализует catalogv1.CatalogServiceServer.
type Server struct {
	catalogv1.UnimplementedCatalogServiceServer

	log            *slog.Logger
	ads            AdProvider
	eventPublisher EventPublisher
}

// NewServer создаёт gRPC-сервер catalog.
func NewServer(log *slog.Logger, ads AdProvider, eventPublisher EventPublisher) *Server {
	return &Server{log: log, ads: ads, eventPublisher: eventPublisher}
}

// GetAd возвращает объявление по ID.
func (s *Server) GetAd(ctx context.Context, req *catalogv1.GetAdRequest) (*catalogv1.AdResponse, error) {
	if req.GetAdId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "ad_id is required")
	}
	ad, err := s.ads.GetAdByID(ctx, req.GetAdId())
	if err != nil {
		if errors.Is(err, adrepo.ErrAdNotFound) {
			return nil, status.Error(codes.NotFound, "ad not found")
		}
		s.log.ErrorContext(ctx, "gRPC GetAd failed",
			slog.Int64("ad_id", req.GetAdId()),
			slog.String("error", err.Error()),
		)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return adToProto(ad), nil
}

// GetAdsByIDs пакетно возвращает объявления. Несуществующие пропускаются.
func (s *Server) GetAdsByIDs(ctx context.Context, req *catalogv1.GetAdsByIDsRequest) (*catalogv1.GetAdsByIDsResponse, error) {
	if len(req.GetAdIds()) == 0 {
		return &catalogv1.GetAdsByIDsResponse{}, nil
	}
	ads := make([]*catalogv1.AdResponse, 0, len(req.GetAdIds()))
	for _, id := range req.GetAdIds() {
		ad, err := s.ads.GetAdByID(ctx, id)
		if err != nil {
			if errors.Is(err, adrepo.ErrAdNotFound) {
				continue
			}
			s.log.ErrorContext(ctx, "gRPC GetAdsByIDs: get failed",
				slog.Int64("ad_id", id),
				slog.String("error", err.Error()),
			)
			continue
		}
		ads = append(ads, adToProto(ad))
	}
	return &catalogv1.GetAdsByIDsResponse{Ads: ads}, nil
}

// CheckAdStatus проверяет, доступно ли объявление для покупки.
// available == true только если статус == "active".
func (s *Server) CheckAdStatus(ctx context.Context, req *catalogv1.CheckAdStatusRequest) (*catalogv1.CheckAdStatusResponse, error) {
	if req.GetAdId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "ad_id is required")
	}
	ad, err := s.ads.GetAdByID(ctx, req.GetAdId())
	if err != nil {
		if errors.Is(err, adrepo.ErrAdNotFound) {
			return nil, status.Error(codes.NotFound, "ad not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &catalogv1.CheckAdStatusResponse{
		Available:     ad.Status == models.AdStatusActive,
		CurrentStatus: ad.Status,
	}, nil
}

// UpdateAdStatus меняет статус объявления (вызывается Commerce при покупке).
// Если новый статус == "sold", публикует событие ad.sold в Kafka.
func (s *Server) UpdateAdStatus(ctx context.Context, req *catalogv1.UpdateAdStatusRequest) (*catalogv1.UpdateAdStatusResponse, error) {
	if req.GetAdId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "ad_id is required")
	}
	if req.GetNewStatus() == "" {
		return nil, status.Error(codes.InvalidArgument, "new_status is required")
	}

	prev, err := s.ads.UpdateAdStatus(ctx, req.GetAdId(), req.GetNewStatus())
	if err != nil {
		if errors.Is(err, adrepo.ErrAdNotFound) {
			return nil, status.Error(codes.NotFound, "ad not found")
		}
		s.log.ErrorContext(ctx, "gRPC UpdateAdStatus failed",
			slog.Int64("ad_id", req.GetAdId()),
			slog.String("new_status", req.GetNewStatus()),
			slog.String("error", err.Error()),
		)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if req.GetNewStatus() == models.AdStatusSold && s.eventPublisher != nil {
		if pubErr := s.eventPublisher.PublishAdSold(ctx, req.GetAdId(), req.GetBuyerId()); pubErr != nil {
			s.log.WarnContext(ctx, "failed to publish ad.sold",
				slog.Int64("ad_id", req.GetAdId()),
				slog.String("error", pubErr.Error()),
			)
		}
	}

	return &catalogv1.UpdateAdStatusResponse{
		Success:        true,
		PreviousStatus: prev,
	}, nil
}

func adToProto(ad models.Ad) *catalogv1.AdResponse {
	return &catalogv1.AdResponse{
		Id:          ad.ID,
		SellerId:    ad.SellerID,
		CategoryId:  ad.CategoryID,
		Title:       ad.Title,
		Description: ad.Description,
		Price:       ad.Price,
		Status:      ad.Status,
		Location:    ad.Location,
		PhotoUrls:   ad.Photos,
		Lat:         ad.Lat,
		Lon:         ad.Lon,
	}
}
