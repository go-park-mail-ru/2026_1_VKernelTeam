package wallet_test

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
	walletrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/wallet"
	walletuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet/mocks"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type topupDeps struct {
	pool     pgxmock.PgxPoolIface
	wallets  *mocks.MockWalletRepo
	payments *mocks.MockPaymentRepo
	provider *mocks.MockPaymentProvider
	uc       *walletuc.Usecase
}

func newTopupDeps(t *testing.T, ctrl *gomock.Controller) *topupDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	walletsMock := mocks.NewMockWalletRepo(ctrl)
	paymentsMock := mocks.NewMockPaymentRepo(ctrl)
	providerMock := mocks.NewMockPaymentProvider(ctrl)
	return &topupDeps{
		pool:     pool,
		wallets:  walletsMock,
		payments: paymentsMock,
		provider: providerMock,
		uc:       walletuc.New(discardLogger(), pool, walletsMock, paymentsMock, providerMock),
	}
}

func TestTopup_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	const userID int64 = 1
	const amount int64 = 500
	const key = "idem-key-1"

	// Idempotency check — ключ ещё не использован.
	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), key).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)

	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any()).
		Return(models.PaymentStatusSucceeded, "mock-ref-1", nil)

	d.pool.ExpectBegin()

	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 0}, nil)

	d.payments.EXPECT().
		InsertTx(gomock.Any(), gomock.Any(), userID, amount, models.PaymentStatusSucceeded, models.PaymentProviderMock, gomock.Any()).
		Return(models.Payment{ID: 42, UserID: userID, Amount: amount, Status: models.PaymentStatusSucceeded}, nil)

	d.wallets.EXPECT().
		InsertTransactionTx(gomock.Any(), gomock.Any(), userID, amount, models.WalletTxTypeTopup, gomock.Any(), gomock.Any()).
		Return(models.WalletTransaction{ID: 100}, nil)

	d.wallets.EXPECT().
		IncrementBalanceTx(gomock.Any(), gomock.Any(), userID, amount).
		Return(int64(500), nil)

	d.pool.ExpectCommit()
	d.pool.ExpectRollback() // defer Rollback после Commit — допустим, pgxmock не требует обязательного matched

	resp, err := d.uc.Topup(context.Background(), userID, amount, key)
	require.NoError(t, err)
	assert.Equal(t, int64(500), resp.Balance)
	assert.Equal(t, int64(42), resp.PaymentID)
}

func TestTopup_InvalidAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	_, err := d.uc.Topup(context.Background(), 1, 0, "key")
	assert.ErrorIs(t, err, walletuc.ErrInvalidAmount)

	_, err = d.uc.Topup(context.Background(), 1, -10, "key")
	assert.ErrorIs(t, err, walletuc.ErrInvalidAmount)

	_, err = d.uc.Topup(context.Background(), 1, 9_999_999, "key")
	assert.ErrorIs(t, err, walletuc.ErrInvalidAmount)
}

func TestTopup_MissingIdempotencyKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	_, err := d.uc.Topup(context.Background(), 1, 100, "")
	assert.ErrorIs(t, err, walletuc.ErrInvalidRequest)
}

func TestTopup_IdempotencyHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	const userID int64 = 7
	const key = "repeat-key"
	paymentID := int64(99)

	existing := models.WalletTransaction{
		ID:          101,
		UserID:      userID,
		Amount:      300,
		Type:        models.WalletTxTypeTopup,
		ReferenceID: &paymentID,
	}
	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), key).
		Return(existing, nil)
	d.wallets.EXPECT().
		Get(gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 300, UpdatedAt: time.Now()}, nil)

	resp, err := d.uc.Topup(context.Background(), userID, 300, key)
	require.NoError(t, err)
	assert.Equal(t, int64(300), resp.Balance)
	assert.Equal(t, paymentID, resp.PaymentID)
}

func TestTopup_ProviderFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), "k").
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any()).
		Return(models.PaymentStatusFailed, "", nil)

	_, err := d.uc.Topup(context.Background(), 1, 100, "k")
	assert.ErrorIs(t, err, walletuc.ErrInvalidRequest)
}

func TestTopup_ProviderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), "k").
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)
	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any()).
		Return("", "", errors.New("network down"))

	_, err := d.uc.Topup(context.Background(), 1, 100, "k")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

func TestGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	d.wallets.EXPECT().
		GetOrCreate(gomock.Any(), int64(5)).
		Return(models.Wallet{UserID: 5, Balance: 123}, nil)

	resp, err := d.uc.GetBalance(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, int64(123), resp.Balance)
	assert.Equal(t, "RUB", resp.Currency)
}

func TestListTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	const userID int64 = 9
	items := []models.WalletTransaction{
		{ID: 30, UserID: userID, Amount: 500, Type: models.WalletTxTypeTopup, CreatedAt: time.Now()},
		{ID: 29, UserID: userID, Amount: -49, Type: models.WalletTxTypePromotionCharge, CreatedAt: time.Now()},
	}
	d.wallets.EXPECT().
		ListTransactionsByUser(gomock.Any(), userID, int64(0), 20).
		Return(items, nil)

	resp, err := d.uc.ListTransactions(context.Background(), userID, 0, 0)
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, int64(500), resp.Items[0].Amount)
	assert.Nil(t, resp.NextCursor)
}

func TestListTransactions_NextCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl)
	defer d.pool.Close()

	const userID int64 = 9
	items := make([]models.WalletTransaction, 0, 20)
	for i := range 20 {
		items = append(items, models.WalletTransaction{ID: int64(i + 1), UserID: userID})
	}
	// items в порядке убывания ID имитируем
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	d.wallets.EXPECT().
		ListTransactionsByUser(gomock.Any(), userID, int64(0), 20).
		Return(items, nil)

	resp, err := d.uc.ListTransactions(context.Background(), userID, 0, 20)
	require.NoError(t, err)
	require.NotNil(t, resp.NextCursor)
	assert.Equal(t, items[len(items)-1].ID, *resp.NextCursor)
}
