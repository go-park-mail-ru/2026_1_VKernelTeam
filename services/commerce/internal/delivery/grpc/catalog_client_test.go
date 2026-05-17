package grpc

import (
	"context"
	"errors"
	"testing"

	catalogv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/catalog/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/grpc/mocks"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const statusActive = "active"

func TestCatalogClient_GetAdByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		GetAd(gomock.Any(), &catalogv1.GetAdRequest{AdId: 1}).
		Return(&catalogv1.AdResponse{Id: 1, SellerId: 7, Title: "X", Price: 100, Status: statusActive}, nil)

	c := &CatalogClient{client: mock}
	ad, err := c.GetAdByID(context.Background(), 1)
	if err != nil || ad.ID != 1 || ad.SellerID != 7 || ad.Status != statusActive {
		t.Fatalf("unexpected: %+v %v", ad, err)
	}
}

func TestCatalogClient_GetAdByID_NotFoundMappedToSentinel(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		GetAd(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	c := &CatalogClient{client: mock}
	_, err := c.GetAdByID(context.Background(), 99)
	if !errors.Is(err, ErrAdNotFound) {
		t.Fatalf("expected ErrAdNotFound, got %v", err)
	}
}

func TestCatalogClient_GetAdByID_GenericError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		GetAd(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "boom"))

	c := &CatalogClient{client: mock}
	_, err := c.GetAdByID(context.Background(), 1)
	if err == nil || errors.Is(err, ErrAdNotFound) {
		t.Fatalf("expected non-sentinel error, got %v", err)
	}
}

func TestCatalogClient_CheckAdStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		CheckAdStatus(gomock.Any(), &catalogv1.CheckAdStatusRequest{AdId: 5}).
		Return(&catalogv1.CheckAdStatusResponse{Available: true, CurrentStatus: statusActive}, nil)

	c := &CatalogClient{client: mock}
	avail, st, err := c.CheckAdStatus(context.Background(), 5)
	if err != nil || !avail || st != statusActive {
		t.Fatalf("unexpected: %v %s %v", avail, st, err)
	}
}

func TestCatalogClient_CheckAdStatus_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		CheckAdStatus(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("rpc fail"))

	c := &CatalogClient{client: mock}
	if _, _, err := c.CheckAdStatus(context.Background(), 5); err == nil {
		t.Fatal("expected error")
	}
}

func TestCatalogClient_UpdateAdStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		UpdateAdStatus(gomock.Any(), &catalogv1.UpdateAdStatusRequest{AdId: 1, NewStatus: "sold", BuyerId: 42}).
		Return(&catalogv1.UpdateAdStatusResponse{Success: true, PreviousStatus: statusActive}, nil)

	c := &CatalogClient{client: mock}
	prev, err := c.UpdateAdStatus(context.Background(), 1, "sold", 42)
	if err != nil || prev != statusActive {
		t.Fatalf("unexpected: %s %v", prev, err)
	}
}

func TestCatalogClient_UpdateAdStatus_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockCatalogServiceClient(ctrl)
	mock.EXPECT().
		UpdateAdStatus(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("rpc fail"))

	c := &CatalogClient{client: mock}
	if _, err := c.UpdateAdStatus(context.Background(), 1, "sold", 42); err == nil {
		t.Fatal("expected error")
	}
}
