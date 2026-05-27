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
	paymentuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
	walletuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/wallet/mocks"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type topupDeps struct {
	pool         pgxmock.PgxPoolIface
	wallets      *mocks.MockWalletRepo
	payments     *mocks.MockPaymentRepo
	provider     *mocks.MockPaymentProvider
	uc           *walletuc.Usecase
	providerName string
}

func newTopupDeps(t *testing.T, ctrl *gomock.Controller, providerName string) *topupDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	walletsMock := mocks.NewMockWalletRepo(ctrl)
	paymentsMock := mocks.NewMockPaymentRepo(ctrl)
	providerMock := mocks.NewMockPaymentProvider(ctrl)
	return &topupDeps{
		pool:         pool,
		wallets:      walletsMock,
		payments:     paymentsMock,
		provider:     providerMock,
		uc:           walletuc.New(discardLogger(), pool, walletsMock, paymentsMock, providerMock, providerName),
		providerName: providerName,
	}
}

// TestTopup_Mock_Succeeded — синхронный mock-провайдер: pending → succeeded в одном вызове.
func TestTopup_Mock_Succeeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
	defer d.pool.Close()

	const userID int64 = 1
	const amount int64 = 500
	const key = "idem-key-1"

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), key).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)

	// 1. TX создания pending-платежа.
	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 0}, nil)
	d.payments.EXPECT().
		InsertTx(gomock.Any(), gomock.Any(), userID, amount, models.PaymentStatusPending, models.PaymentProviderMock, gomock.Any()).
		Return(models.Payment{ID: 42, UserID: userID, Amount: amount, Status: models.PaymentStatusPending}, nil)
	d.pool.ExpectCommit()

	// 2. Вызов провайдера.
	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any(), key).
		Return(paymentuc.InitResult{
			Status:      models.PaymentStatusSucceeded,
			ProviderRef: "mock-ref-1",
		}, nil)

	// 3. TX сохранения provider_ref.
	d.pool.ExpectBegin()
	d.payments.EXPECT().
		UpdateProviderRefTx(gomock.Any(), gomock.Any(), int64(42), "mock-ref-1", "", gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	// 4. TX зачисления баланса (applySucceeded).
	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 0}, nil)
	d.wallets.EXPECT().
		InsertTransactionTx(gomock.Any(), gomock.Any(), userID, amount, models.WalletTxTypeTopup, gomock.Any(), gomock.Any()).
		Return(models.WalletTransaction{ID: 100}, nil)
	d.wallets.EXPECT().
		IncrementBalanceTx(gomock.Any(), gomock.Any(), userID, amount).
		Return(int64(500), nil)
	d.payments.EXPECT().
		UpdateStatusTx(gomock.Any(), gomock.Any(), int64(42), models.PaymentStatusSucceeded, gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	resp, err := d.uc.Topup(context.Background(), userID, amount, key)
	require.NoError(t, err)
	assert.Equal(t, int64(500), resp.Balance)
	assert.Equal(t, int64(42), resp.PaymentID)
	assert.Equal(t, models.PaymentStatusSucceeded, resp.Status)
	assert.Empty(t, resp.ConfirmationURL)
}

// TestTopup_YooKassa_Pending — асинхронный провайдер: pending + confirmation_url, баланс НЕ меняется.
func TestTopup_YooKassa_Pending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	const userID int64 = 7
	const amount int64 = 1000
	const key = "yoo-key"

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), key).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)

	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID, Balance: 0}, nil)
	d.payments.EXPECT().
		InsertTx(gomock.Any(), gomock.Any(), userID, amount, models.PaymentStatusPending, models.PaymentProviderYooKassa, gomock.Any()).
		Return(models.Payment{ID: 77, UserID: userID, Amount: amount, Status: models.PaymentStatusPending}, nil)
	d.pool.ExpectCommit()

	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any(), key).
		Return(paymentuc.InitResult{
			Status:          models.PaymentStatusPending,
			ProviderRef:     "yoo-uuid-1",
			ConfirmationURL: "https://yoo/confirm/123",
		}, nil)

	d.pool.ExpectBegin()
	d.payments.EXPECT().
		UpdateProviderRefTx(gomock.Any(), gomock.Any(), int64(77), "yoo-uuid-1", "https://yoo/confirm/123", gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	resp, err := d.uc.Topup(context.Background(), userID, amount, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Balance) // pending — баланс не зачислен
	assert.Equal(t, int64(77), resp.PaymentID)
	assert.Equal(t, models.PaymentStatusPending, resp.Status)
	assert.Equal(t, "https://yoo/confirm/123", resp.ConfirmationURL)
}

func TestTopup_InvalidAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
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
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
	defer d.pool.Close()

	_, err := d.uc.Topup(context.Background(), 1, 100, "")
	assert.ErrorIs(t, err, walletuc.ErrInvalidRequest)
}

func TestTopup_IdempotencyHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
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
	assert.Equal(t, models.PaymentStatusSucceeded, resp.Status)
}

func TestTopup_ProviderError_MarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	const userID int64 = 1
	const amount int64 = 100
	const key = "k"

	d.wallets.EXPECT().
		GetTransactionByIdempotencyKey(gomock.Any(), key).
		Return(models.WalletTransaction{}, walletrepo.ErrTransactionNotFound)

	d.pool.ExpectBegin()
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{}, nil)
	d.payments.EXPECT().
		InsertTx(gomock.Any(), gomock.Any(), userID, amount, models.PaymentStatusPending, models.PaymentProviderYooKassa, gomock.Any()).
		Return(models.Payment{ID: 1, UserID: userID, Amount: amount}, nil)
	d.pool.ExpectCommit()

	d.provider.EXPECT().
		InitPayment(gomock.Any(), gomock.Any(), key).
		Return(paymentuc.InitResult{}, errors.New("network down"))

	d.pool.ExpectBegin()
	d.payments.EXPECT().
		UpdateStatusTx(gomock.Any(), gomock.Any(), int64(1), models.PaymentStatusFailed, gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	_, err := d.uc.Topup(context.Background(), userID, amount, key)
	assert.Error(t, err)
}

func TestApplyProviderUpdate_Succeeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	const userID int64 = 5
	const amount int64 = 200
	providerRef := "yoo-uuid-7"

	d.pool.ExpectBegin()
	d.payments.EXPECT().
		GetByProviderRefForUpdateTx(gomock.Any(), gomock.Any(), models.PaymentProviderYooKassa, providerRef).
		Return(models.Payment{
			ID: 50, UserID: userID, Amount: amount,
			Status: models.PaymentStatusPending, Provider: models.PaymentProviderYooKassa,
			ProviderRef: &providerRef,
		}, nil)
	d.wallets.EXPECT().
		GetForUpdateTx(gomock.Any(), gomock.Any(), userID).
		Return(models.Wallet{UserID: userID}, nil)
	d.wallets.EXPECT().
		InsertTransactionTx(gomock.Any(), gomock.Any(), userID, amount, models.WalletTxTypeTopup, gomock.Any(), gomock.Any()).
		Return(models.WalletTransaction{ID: 555}, nil)
	d.wallets.EXPECT().
		IncrementBalanceTx(gomock.Any(), gomock.Any(), userID, amount).
		Return(int64(amount), nil)
	d.payments.EXPECT().
		UpdateStatusTx(gomock.Any(), gomock.Any(), int64(50), models.PaymentStatusSucceeded, gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	err := d.uc.ApplyProviderUpdate(context.Background(), models.PaymentProviderYooKassa, providerRef, models.PaymentStatusSucceeded, []byte(`{}`))
	require.NoError(t, err)
}

// Повторный webhook на уже succeeded-платеже — no-op, без побочных эффектов.
func TestApplyProviderUpdate_AlreadyApplied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	providerRef := "yoo-uuid-9"

	d.pool.ExpectBegin()
	d.payments.EXPECT().
		GetByProviderRefForUpdateTx(gomock.Any(), gomock.Any(), models.PaymentProviderYooKassa, providerRef).
		Return(models.Payment{
			ID: 1, Status: models.PaymentStatusSucceeded, Provider: models.PaymentProviderYooKassa,
			ProviderRef: &providerRef,
		}, nil)
	d.pool.ExpectRollback()

	err := d.uc.ApplyProviderUpdate(context.Background(), models.PaymentProviderYooKassa, providerRef, models.PaymentStatusSucceeded, nil)
	require.NoError(t, err)
}

func TestApplyProviderUpdate_Cancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	providerRef := "yoo-uuid-x"

	d.pool.ExpectBegin()
	d.payments.EXPECT().
		GetByProviderRefForUpdateTx(gomock.Any(), gomock.Any(), models.PaymentProviderYooKassa, providerRef).
		Return(models.Payment{
			ID: 33, Status: models.PaymentStatusPending, Provider: models.PaymentProviderYooKassa,
			ProviderRef: &providerRef,
		}, nil)
	d.payments.EXPECT().
		UpdateStatusTx(gomock.Any(), gomock.Any(), int64(33), models.PaymentStatusCancelled, gomock.Any()).
		Return(nil)
	d.pool.ExpectCommit()

	err := d.uc.ApplyProviderUpdate(context.Background(), models.PaymentProviderYooKassa, providerRef, models.PaymentStatusCancelled, nil)
	require.NoError(t, err)
}

func TestGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
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
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
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
	d := newTopupDeps(t, ctrl, models.PaymentProviderMock)
	defer d.pool.Close()

	const userID int64 = 9
	items := make([]models.WalletTransaction, 0, 20)
	for i := range 20 {
		items = append(items, models.WalletTransaction{ID: int64(i + 1), UserID: userID})
	}
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

func TestGetPaymentStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	url := "https://yoo/confirm"
	d.payments.EXPECT().
		GetByID(gomock.Any(), int64(11)).
		Return(models.Payment{
			ID: 11, UserID: 4, Amount: 200, Status: models.PaymentStatusPending,
			ConfirmationURL: &url,
		}, nil)

	resp, err := d.uc.GetPaymentStatus(context.Background(), 4, 11)
	require.NoError(t, err)
	assert.Equal(t, int64(11), resp.PaymentID)
	assert.Equal(t, models.PaymentStatusPending, resp.Status)
	assert.Equal(t, url, resp.ConfirmationURL)
}

func TestGetPaymentStatus_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := newTopupDeps(t, ctrl, models.PaymentProviderYooKassa)
	defer d.pool.Close()

	d.payments.EXPECT().
		GetByID(gomock.Any(), int64(11)).
		Return(models.Payment{ID: 11, UserID: 99}, nil)

	_, err := d.uc.GetPaymentStatus(context.Background(), 4, 11)
	assert.ErrorIs(t, err, walletuc.ErrPaymentNotFound)
}
