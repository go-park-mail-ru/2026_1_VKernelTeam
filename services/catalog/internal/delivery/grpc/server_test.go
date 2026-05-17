package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	catalogv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/catalog/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/grpc/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	adrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const statusSold = "sold"

// setup поднимает gomock-controller, моки и Server.
func setup(t *testing.T) (*Server, *mocks.MockAdProvider, *mocks.MockEventPublisher) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	ad := mocks.NewMockAdProvider(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)), ad, pub)
	return srv, ad, pub
}

func TestGetAd_Success(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).
		Return(models.Ad{ID: 1, Title: "X", Price: 100, Status: "active"}, nil)

	resp, err := srv.GetAd(context.Background(), &catalogv1.GetAdRequest{AdId: 1})
	if err != nil || resp.GetTitle() != "X" {
		t.Fatalf("unexpected: %v %v", resp, err)
	}
}

func TestGetAd_NotFound(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(99)).
		Return(models.Ad{}, adrepo.ErrAdNotFound)

	_, err := srv.GetAd(context.Background(), &catalogv1.GetAdRequest{AdId: 99})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestGetAd_InvalidID(t *testing.T) {
	srv, _, _ := setup(t)
	_, err := srv.GetAd(context.Background(), &catalogv1.GetAdRequest{AdId: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetAd_InternalError(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).
		Return(models.Ad{}, errors.New("db down"))

	_, err := srv.GetAd(context.Background(), &catalogv1.GetAdRequest{AdId: 1})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestGetAdsByIDs_EmptyReturnsEmpty(t *testing.T) {
	srv, _, _ := setup(t)
	resp, err := srv.GetAdsByIDs(context.Background(), &catalogv1.GetAdsByIDsRequest{})
	if err != nil || len(resp.GetAds()) != 0 {
		t.Fatalf("expected empty resp, got %v %v", resp, err)
	}
}

func TestGetAdsByIDs_SkipsNotFoundAndErrors(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(models.Ad{}, adrepo.ErrAdNotFound)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(2)).Return(models.Ad{}, errors.New("db blip"))
	ad.EXPECT().GetAdByID(gomock.Any(), int64(3)).Return(models.Ad{ID: 3, Title: "Z"}, nil)

	resp, err := srv.GetAdsByIDs(context.Background(), &catalogv1.GetAdsByIDsRequest{AdIds: []int64{1, 2, 3}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.GetAds()) != 1 || resp.GetAds()[0].GetId() != 3 {
		t.Fatalf("expected only id=3, got %+v", resp.GetAds())
	}
}

func TestCheckAdStatus_InvalidID(t *testing.T) {
	srv, _, _ := setup(t)
	_, err := srv.CheckAdStatus(context.Background(), &catalogv1.CheckAdStatusRequest{AdId: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestCheckAdStatus_Available(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(models.Ad{Status: "active"}, nil)

	resp, err := srv.CheckAdStatus(context.Background(), &catalogv1.CheckAdStatusRequest{AdId: 1})
	if err != nil || !resp.GetAvailable() {
		t.Fatalf("expected available, got %v %v", resp, err)
	}
}

func TestCheckAdStatus_NotAvailableWhenNotActive(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(models.Ad{Status: statusSold}, nil)

	resp, err := srv.CheckAdStatus(context.Background(), &catalogv1.CheckAdStatusRequest{AdId: 1})
	if err != nil || resp.GetAvailable() || resp.GetCurrentStatus() != statusSold {
		t.Fatalf("unexpected: %v %v", resp, err)
	}
}

func TestCheckAdStatus_NotFound(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(models.Ad{}, adrepo.ErrAdNotFound)

	_, err := srv.CheckAdStatus(context.Background(), &catalogv1.CheckAdStatusRequest{AdId: 1})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestUpdateAdStatus_InvalidID(t *testing.T) {
	srv, _, _ := setup(t)
	_, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{NewStatus: statusSold})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdateAdStatus_EmptyNewStatus(t *testing.T) {
	srv, _, _ := setup(t)
	_, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{AdId: 1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdateAdStatus_NotFound(t *testing.T) {
	srv, ad, _ := setup(t)
	ad.EXPECT().UpdateAdStatus(gomock.Any(), int64(1), statusSold).
		Return("", adrepo.ErrAdNotFound)

	_, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{AdId: 1, NewStatus: statusSold})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestUpdateAdStatus_PublishesAdSold(t *testing.T) {
	srv, ad, pub := setup(t)
	ad.EXPECT().UpdateAdStatus(gomock.Any(), int64(1), statusSold).Return("active", nil)
	pub.EXPECT().PublishAdSold(gomock.Any(), int64(1), int64(42)).Return(nil)

	_, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{
		AdId: 1, NewStatus: statusSold, BuyerId: 42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAdStatus_NotSold_DoesNotPublish(t *testing.T) {
	srv, ad, pub := setup(t)
	ad.EXPECT().UpdateAdStatus(gomock.Any(), int64(1), "reserved").Return("active", nil)
	// pub.EXPECT() не выставлен => mock упадёт, если PublishAdSold позовут.
	_ = pub

	_, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{
		AdId: 1, NewStatus: "reserved",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestUpdateAdStatus_NilPublisherIsOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	ad := mocks.NewMockAdProvider(ctrl)
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)), ad, nil)
	ad.EXPECT().UpdateAdStatus(gomock.Any(), int64(1), statusSold).Return("active", nil)

	resp, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{
		AdId: 1, NewStatus: statusSold,
	})
	if err != nil || !resp.GetSuccess() {
		t.Fatalf("unexpected: %v %v", resp, err)
	}
}

func TestUpdateAdStatus_PublishErrorDoesNotFailRPC(t *testing.T) {
	srv, ad, pub := setup(t)
	ad.EXPECT().UpdateAdStatus(gomock.Any(), int64(1), statusSold).Return("active", nil)
	pub.EXPECT().PublishAdSold(gomock.Any(), int64(1), int64(5)).Return(errors.New("kafka down"))

	resp, err := srv.UpdateAdStatus(context.Background(), &catalogv1.UpdateAdStatusRequest{
		AdId: 1, NewStatus: statusSold, BuyerId: 5,
	})
	if err != nil || !resp.GetSuccess() {
		t.Fatalf("publish error must not break RPC: %v %v", resp, err)
	}
}
