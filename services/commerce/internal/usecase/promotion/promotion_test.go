package promotion_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	promorepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/promotion"
	walletrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/wallet"
	promouc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/promotion"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/promotion/mocks"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type deps struct {
	pool       pgxmock.PgxPoolIface
	plans      *mocks.MockPlanRepo
	promotions *mocks.MockPromotionRepo
	wallets    *mocks.MockWalletRepo
	cache      *mocks.MockPlanCache
	ads        *mocks.MockAdsProvider
	uc         *promouc.Usecase
}

func newDeps(t *testing.T, ctrl *gomock.Controller) *deps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	plansMock := mocks.NewMockPlanRepo(ctrl)
	promotionsMock := mocks.NewMockPromotionRepo(ctrl)
	walletsMock := mocks.NewMockWalletRepo(ctrl)
	cacheMock := mocks.NewMockPlanCache(ctrl)
	adsMock := mocks.NewMockAdsProvider(ctrl)
	return &deps{
		pool:       pool,
		plans:      plansMock,
		promotions: promotionsMock,
		wallets:    walletsMock,
		cache:      cacheMock,
		ads:        adsMock,
		uc: promouc.New(
			discardLogger(),
			pool,
			plansMock, promotionsMock, walletsMock, cacheMock, adsMock,
		),
	}
}

const (
	userID    = int64(1)
	productID = int64(10)
	planID    = int64(7)
	planCode  = "boost_7d"
	planPrice = int64(199)
	idemKey   = "uuid-key-1"
)

func basePlan() models.PromotionPlan {
	return models.PromotionPlan{
		ID: planID, Code: planCode, Kind: models.PromotionKindBoost,
		DurationDays: 7, Price: planPrice, IsActive: true,
	}
}

func ownerAd(status string) models.Ad {
	return models.Ad{ID: productID, SellerID: userID, Status: status}
}

func TestPurchase_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).Return(ownerAd(models.AdStatusActive), nil)

	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 1000}, nil)
	d.promotions.EXPECT().
		MaxActiveExpiresTx(gomock.Any(), gomock.Any(), productID, models.PromotionKindBoost).
		Return(time.Time{}, false, nil)
	d.promotions.EXPECT().
		InsertTx(gomock.Any(), gomock.Any(), productID, userID, planID, models.PromotionKindBoost, gomock.Any(), planPrice).
		Return(models.Promotion{
			ID:        555,
			ProductID: productID,
			UserID:    userID,
			PlanID:    planID,
			Kind:      models.PromotionKindBoost,
			StartsAt:  time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			PricePaid: planPrice,
		}, nil)
	d.wallets.EXPECT().
		InsertTransactionTx(gomock.Any(), gomock.Any(), userID, -planPrice, models.WalletTxTypePromotionCharge, gomock.Any(), gomock.Any()).
		Return(models.WalletTransaction{ID: 90}, nil)
	d.wallets.EXPECT().
		DecrementBalanceTx(gomock.Any(), gomock.Any(), userID, planPrice).
		Return(int64(801), nil)
	d.pool.ExpectCommit()
	d.pool.ExpectRollback()

	resp, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	require.NoError(t, err)
	assert.Equal(t, int64(555), resp.Promotion.ID)
	assert.Equal(t, models.PromotionKindBoost, resp.Promotion.Kind)
	assert.Equal(t, planCode, resp.Promotion.PlanCode)
	assert.Equal(t, int64(801), resp.WalletBalance)
}

func TestPurchase_PlanNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), "missing").
		Return(models.PromotionPlan{}, promorepo.ErrPlanNotFound)

	_, err := d.uc.Purchase(context.Background(), userID, productID, "missing", idemKey)
	assert.ErrorIs(t, err, promouc.ErrPlanNotFound)
}

func TestPurchase_PlanInactive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).
		Return(basePlan(), promorepo.ErrPlanInactive)

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrPlanInactive)
}

func TestPurchase_AdNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).
		Return(models.Ad{}, errors.New("not found"))

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrAdNotFound)
}

func TestPurchase_NotOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	stranger := models.Ad{ID: productID, SellerID: 999, Status: models.AdStatusActive}
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).Return(stranger, nil)

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrNotAdOwner)
}

func TestPurchase_InvalidStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).Return(ownerAd(models.AdStatusSold), nil)

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrInvalidAdStatus)
}

func TestPurchase_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).Return(ownerAd(models.AdStatusActive), nil)

	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 50}, nil)
	d.pool.ExpectRollback()

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrInsufficientFund)
}

func TestPurchase_Extension(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	// Уже есть активный boost до now+3d, продлеваем ещё на 7d → итого до now+10d.
	existingExpires := time.Now().Add(3 * 24 * time.Hour).UTC()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.plans.EXPECT().GetByCode(gomock.Any(), planCode).Return(basePlan(), nil)
	d.ads.EXPECT().GetAdByID(gomock.Any(), productID).Return(ownerAd(models.AdStatusActive), nil)

	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 1000}, nil)
	d.promotions.EXPECT().
		MaxActiveExpiresTx(gomock.Any(), gomock.Any(), productID, models.PromotionKindBoost).
		Return(existingExpires, true, nil)

	// Проверяем, что expires рассчитан от existingExpires, а не от now.
	expectedExpires := existingExpires.Add(7 * 24 * time.Hour)
	d.promotions.EXPECT().
		InsertTx(
			gomock.Any(), gomock.Any(),
			productID, userID, planID, models.PromotionKindBoost,
			gomock.AssignableToTypeOf(time.Time{}), planPrice,
		).
		DoAndReturn(func(_ context.Context, _ any, _, _, _ int64, _ string, expiresAt time.Time, _ int64) (models.Promotion, error) {
			delta := expiresAt.Sub(expectedExpires).Abs()
			require.LessOrEqual(t, delta, time.Second, "expiresAt должен быть == existingExpires + 7d")
			return models.Promotion{
				ID:        777,
				ProductID: productID,
				PlanID:    planID,
				Kind:      models.PromotionKindBoost,
				ExpiresAt: expiresAt,
				PricePaid: planPrice,
			}, nil
		})
	d.wallets.EXPECT().
		InsertTransactionTx(gomock.Any(), gomock.Any(), userID, -planPrice, models.WalletTxTypePromotionCharge, gomock.Any(), gomock.Any()).
		Return(models.WalletTransaction{ID: 33}, nil)
	d.wallets.EXPECT().
		DecrementBalanceTx(gomock.Any(), gomock.Any(), userID, planPrice).
		Return(int64(801), nil)
	d.pool.ExpectCommit()
	d.pool.ExpectRollback()

	resp, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	require.NoError(t, err)
	assert.Equal(t, int64(777), resp.Promotion.ID)
}

func TestPurchase_IdempotencyHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	promoID := int64(123)
	existing := models.WalletTransaction{
		ID: 9, UserID: userID, Amount: -planPrice,
		Type: models.WalletTxTypePromotionCharge, ReferenceID: &promoID,
	}
	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(existing, nil)
	d.promotions.EXPECT().GetByID(gomock.Any(), promoID).
		Return(models.Promotion{
			ID: promoID, ProductID: productID, PlanID: planID,
			Kind: models.PromotionKindBoost, PricePaid: planPrice,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}, nil)
	d.plans.EXPECT().GetByID(gomock.Any(), planID).Return(basePlan(), nil)
	d.wallets.EXPECT().Get(gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 600}, nil)

	resp, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	require.NoError(t, err)
	assert.Equal(t, promoID, resp.Promotion.ID)
	assert.Equal(t, int64(600), resp.WalletBalance)
}

func TestPurchase_IdempotencyKey_UsedForDifferentOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	// Ключ уже использован для пополнения — конфликт.
	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), idemKey).
		Return(models.WalletTransaction{
			ID: 9, Type: models.WalletTxTypeTopup, ReferenceID: nil,
		}, nil)

	_, err := d.uc.Purchase(context.Background(), userID, productID, planCode, idemKey)
	assert.ErrorIs(t, err, promouc.ErrInvalidRequest)
}

func TestPurchase_MissingFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	_, err := d.uc.Purchase(context.Background(), userID, productID, "", idemKey)
	assert.ErrorIs(t, err, promouc.ErrInvalidRequest)

	_, err = d.uc.Purchase(context.Background(), userID, productID, planCode, "")
	assert.ErrorIs(t, err, promouc.ErrInvalidRequest)
}

func TestGetPlans_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	cached := []models.PromotionPlan{basePlan()}
	d.cache.EXPECT().Get(gomock.Any()).Return(cached, nil)

	plans, err := d.uc.GetPlans(context.Background())
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, planCode, plans[0].Code)
}

func TestGetPlans_CacheMiss_FetchAndStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.cache.EXPECT().Get(gomock.Any()).Return(nil, errors.New("miss"))
	d.plans.EXPECT().GetActivePlans(gomock.Any()).
		Return([]models.PromotionPlan{basePlan()}, nil)
	d.cache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

	plans, err := d.uc.GetPlans(context.Background())
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, planID, plans[0].ID)
	assert.Equal(t, planPrice, plans[0].Price)
}

func TestListActiveByAd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.promotions.EXPECT().GetActiveByAd(gomock.Any(), productID).
		Return([]models.Promotion{
			{ID: 1, ProductID: productID, PlanID: planID, Kind: models.PromotionKindBoost},
		}, nil)
	d.plans.EXPECT().GetByID(gomock.Any(), planID).Return(basePlan(), nil)

	items, err := d.uc.ListActiveByAd(context.Background(), productID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, planCode, items[0].PlanCode)
}

func TestListByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newDeps(t, ctrl)
	defer d.pool.Close()

	d.promotions.EXPECT().ListByUser(gomock.Any(), userID, int64(0), 20).
		Return([]models.Promotion{
			{ID: 2, PlanID: planID, Kind: models.PromotionKindBoost},
		}, nil)
	d.plans.EXPECT().GetByID(gomock.Any(), planID).Return(basePlan(), nil)

	resp, err := d.uc.ListByUser(context.Background(), userID, 0, 0)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Nil(t, resp.NextCursor)
}
