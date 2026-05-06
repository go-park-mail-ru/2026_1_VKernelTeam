package grpc

//go:generate mockgen -destination=mocks/mock_catalog_grpc.go -package=mocks -source=../../../../../proto/gen/catalog/v1/catalog_grpc.pb.go

import (
	"context"
	"fmt"

	catalogv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/catalog/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ErrAdNotFound - gRPC NotFound от catalog приведён к локальной sentinel-ошибке,
// чтобы commerce-юскейсы могли её матчить через errors.Is.
type adNotFoundError struct{}

func (adNotFoundError) Error() string { return "ad not found" }

// ErrAdNotFound публикует sentinel.
var ErrAdNotFound = adNotFoundError{}

// CatalogClient - обёртка над gRPC CatalogService.
// Реализует интерфейсы AdsProvider/AdProvider, нужные usecase'ам cart и chat
// (они ждут метод GetAdByID(ctx, id) (models.Ad, error)).
type CatalogClient struct {
	client catalogv1.CatalogServiceClient
	conn   *grpclib.ClientConn
}

// NewCatalogClient подключается к Catalog gRPC.
func NewCatalogClient(addr string) (*CatalogClient, error) {
	conn, err := grpclib.NewClient(addr, grpclib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc.NewCatalogClient: %w", err)
	}
	return &CatalogClient{client: catalogv1.NewCatalogServiceClient(conn), conn: conn}, nil
}

// Close закрывает соединение.
func (c *CatalogClient) Close() error { return c.conn.Close() }

// GetAdByID реализует интерфейс AdProvider/AdsProvider из usecase'ов commerce.
// Конвертирует proto-AdResponse в локальный slim-models.Ad.
func (c *CatalogClient) GetAdByID(ctx context.Context, id int64) (models.Ad, error) {
	resp, err := c.client.GetAd(ctx, &catalogv1.GetAdRequest{AdId: id})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return models.Ad{}, ErrAdNotFound
		}
		return models.Ad{}, fmt.Errorf("catalog.GetAd: %w", err)
	}
	return models.Ad{
		ID:       resp.GetId(),
		SellerID: resp.GetSellerId(),
		Title:    resp.GetTitle(),
		Price:    resp.GetPrice(),
		Status:   resp.GetStatus(),
	}, nil
}

// CheckAdStatus вернёт true если объявление сейчас "active".
func (c *CatalogClient) CheckAdStatus(ctx context.Context, id int64) (available bool, currentStatus string, err error) {
	resp, err := c.client.CheckAdStatus(ctx, &catalogv1.CheckAdStatusRequest{AdId: id})
	if err != nil {
		return false, "", fmt.Errorf("catalog.CheckAdStatus: %w", err)
	}
	return resp.GetAvailable(), resp.GetCurrentStatus(), nil
}

// UpdateAdStatus меняет статус объявления (вызывается при подтверждении покупки).
func (c *CatalogClient) UpdateAdStatus(ctx context.Context, adID int64, newStatus string, buyerID int64) (prev string, err error) {
	resp, err := c.client.UpdateAdStatus(ctx, &catalogv1.UpdateAdStatusRequest{
		AdId:      adID,
		NewStatus: newStatus,
		BuyerId:   buyerID,
	})
	if err != nil {
		return "", fmt.Errorf("catalog.UpdateAdStatus: %w", err)
	}
	return resp.GetPreviousStatus(), nil
}
