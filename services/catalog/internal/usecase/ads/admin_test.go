package ads

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/usecase/ads/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newAdminUC(t *testing.T) (
	*Ads,
	*mocks.MockAdsProvider,
	*mocks.MockSystemMessenger,
	*mocks.MockModerationGate,
	*gomock.Controller,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockAdsProvider(ctrl)
	sys := mocks.NewMockSystemMessenger(ctrl)
	gate := mocks.NewMockModerationGate(ctrl)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := New(log, storage, nil, config.SearchConfig{}, nil).WithAdmin(sys, gate, 42)
	return uc, storage, sys, gate, ctrl
}

func TestAds_AdminDeleteAd_Success(t *testing.T) {
	uc, storage, sys, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	storage.EXPECT().
		AdminDeleteAd(gomock.Any(), int64(10), int64(1)).
		Return(int64(7), "Велосипед", nil)
	sys.EXPECT().
		Send(gomock.Any(), int64(42), int64(7), int64(10), gomock.Any()).
		Return(nil)

	assert.NoError(t, uc.AdminDeleteAd(context.Background(), 10, 1))
}

func TestAds_AdminDeleteAd_RepoError(t *testing.T) {
	uc, storage, _, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	want := errors.New("not found")
	storage.EXPECT().AdminDeleteAd(gomock.Any(), int64(10), int64(1)).Return(int64(0), "", want)

	assert.ErrorIs(t, uc.AdminDeleteAd(context.Background(), 10, 1), want)
}

func TestAds_ApproveAd_Success(t *testing.T) {
	uc, storage, sys, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	storage.EXPECT().
		ModerateAd(gomock.Any(), int64(11), models.AdStatusActive, "").
		Return(int64(8), "Книга", nil)
	sys.EXPECT().Send(gomock.Any(), int64(42), int64(8), int64(11), gomock.Any()).Return(nil)

	assert.NoError(t, uc.ApproveAd(context.Background(), 11, 1))
}

func TestAds_RejectAd_WithReason(t *testing.T) {
	uc, storage, sys, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	storage.EXPECT().
		ModerateAd(gomock.Any(), int64(12), models.AdStatusRejected, "spam").
		Return(int64(9), "Картинка", nil)
	sys.EXPECT().Send(gomock.Any(), int64(42), int64(9), int64(12), gomock.Any()).Return(nil)

	assert.NoError(t, uc.RejectAd(context.Background(), 12, 1, "spam"))
}

func TestAds_RejectAd_NoReason(t *testing.T) {
	uc, storage, sys, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	storage.EXPECT().
		ModerateAd(gomock.Any(), int64(13), models.AdStatusRejected, "").
		Return(int64(9), "Картинка", nil)
	sys.EXPECT().Send(gomock.Any(), int64(42), int64(9), int64(13), gomock.Any()).Return(nil)

	assert.NoError(t, uc.RejectAd(context.Background(), 13, 1, ""))
}

func TestAds_GetModerationQueue(t *testing.T) {
	uc, storage, _, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	want := []models.Ad{{ID: 1}, {ID: 2}}
	storage.EXPECT().GetModerationQueue(gomock.Any()).Return(want, nil)

	got, err := uc.GetModerationQueue(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAds_GetUserAdsByStatus(t *testing.T) {
	uc, storage, _, _, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	want := []models.Ad{{ID: 3}}
	storage.EXPECT().GetUserAdsByStatus(gomock.Any(), int64(5), "pending_moderation").Return(want, nil)

	got, err := uc.GetUserAdsByStatus(context.Background(), 5, "pending_moderation")
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAds_IsModerationEnabled(t *testing.T) {
	uc, _, _, gate, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	gate.EXPECT().IsEnabled(gomock.Any()).Return(true)
	assert.True(t, uc.IsModerationEnabled(context.Background()))
}

func TestAds_IsModerationEnabled_NoGate(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := New(log, nil, nil, config.SearchConfig{}, nil)
	assert.False(t, uc.IsModerationEnabled(context.Background()))
}

func TestAds_SetModerationEnabled(t *testing.T) {
	uc, _, _, gate, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	gate.EXPECT().SetEnabled(gomock.Any(), true, int64(1)).Return(nil)
	assert.NoError(t, uc.SetModerationEnabled(context.Background(), true, 1))
}

func TestAds_SetModerationEnabled_NoGate(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := New(log, nil, nil, config.SearchConfig{}, nil)
	assert.ErrorIs(t, uc.SetModerationEnabled(context.Background(), true, 1), ErrAdminDepsNotConfigured)
}

func TestAds_CreateAd_ModerationOn_SwitchesStatusAndNotifies(t *testing.T) {
	uc, storage, sys, gate, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	gate.EXPECT().IsEnabled(gomock.Any()).Return(true)
	storage.EXPECT().
		CreateAd(gomock.Any(), gomock.AssignableToTypeOf(&dto.CreateAdRequest{})).
		DoAndReturn(func(_ context.Context, req *dto.CreateAdRequest) (int64, error) {
			assert.Equal(t, models.AdStatusPendingModeration, req.Status)
			return 100, nil
		})
	sys.EXPECT().Send(gomock.Any(), int64(42), int64(77), int64(100), gomock.Any()).Return(nil)

	req := &dto.CreateAdRequest{UserID: 77, Title: "T", Status: models.AdStatusActive}
	id, err := uc.CreateAd(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), id)
}

func TestAds_CreateAd_ModerationOff_KeepsStatus(t *testing.T) {
	uc, storage, _, gate, ctrl := newAdminUC(t)
	defer ctrl.Finish()

	gate.EXPECT().IsEnabled(gomock.Any()).Return(false)
	storage.EXPECT().
		CreateAd(gomock.Any(), gomock.AssignableToTypeOf(&dto.CreateAdRequest{})).
		DoAndReturn(func(_ context.Context, req *dto.CreateAdRequest) (int64, error) {
			assert.Equal(t, models.AdStatusActive, req.Status)
			return 101, nil
		})

	req := &dto.CreateAdRequest{UserID: 77, Title: "T", Status: models.AdStatusActive}
	id, err := uc.CreateAd(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, int64(101), id)
}
