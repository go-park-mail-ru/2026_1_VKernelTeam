// Package wallet — usecase внутреннего кошелька: пополнение и просмотр баланса/истории.
package wallet

//go:generate mockgen -source=wallet.go -destination=mocks/mock_wallet.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	commercemetrics "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/metrics"
	commercepostgres "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/postgres"
)

const (
	opWalletTopup        = "usecase.wallet.Topup"
	opWalletGetBalance   = "usecase.wallet.GetBalance"
	opWalletListTxs      = "usecase.wallet.ListTransactions"
	currencyRUB          = "RUB"
	defaultListLimit     = 20
	maxListLimit         = 100
	maxTopupAmountRubles = 1_000_000
)

// Бизнес-ошибки usecase.
var (
	ErrInvalidAmount  = errors.New("INVALID_AMOUNT")
	ErrInvalidRequest = errors.New("INVALID_REQUEST")
	ErrAmountTooLarge = errors.New("INVALID_AMOUNT")
)

// TxRunner — узкий контракт пула: открыть транзакцию.
type TxRunner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// WalletRepo — репозиторий кошельков, нужный usecase'у.
type WalletRepo interface {
	Get(ctx context.Context, userID int64) (models.Wallet, error)
	GetOrCreate(ctx context.Context, userID int64) (models.Wallet, error)
	GetForUpdateTx(ctx context.Context, tx pgx.Tx, userID int64) (models.Wallet, error)
	IncrementBalanceTx(ctx context.Context, tx pgx.Tx, userID, amount int64) (int64, error)
	DecrementBalanceTx(ctx context.Context, tx pgx.Tx, userID, amount int64) (int64, error)
	InsertTransactionTx(
		ctx context.Context, tx pgx.Tx,
		userID, amount int64, txType string, referenceID *int64, idempotencyKey *string,
	) (models.WalletTransaction, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (models.WalletTransaction, error)
	ListTransactionsByUser(ctx context.Context, userID int64, cursor int64, limit int) ([]models.WalletTransaction, error)
}

// PaymentRepo — репозиторий платежей.
type PaymentRepo interface {
	InsertTx(
		ctx context.Context, tx pgx.Tx,
		userID, amount int64, status, provider string, providerRef *string,
	) (models.Payment, error)
}

// PaymentProvider — интерфейс провайдера платежей (см. usecase/payment).
type PaymentProvider interface {
	InitPayment(ctx context.Context, p models.Payment) (status string, providerRef string, err error)
}

// Usecase реализует Topup и чтение кошелька.
type Usecase struct {
	log      *slog.Logger
	tx       TxRunner
	wallets  WalletRepo
	payments PaymentRepo
	provider PaymentProvider
}

// New создаёт usecase кошелька с зависимостями репозиториев и провайдера платежей.
func New(
	log *slog.Logger,
	tx TxRunner,
	wallets WalletRepo,
	payments PaymentRepo,
	provider PaymentProvider,
) *Usecase {
	return &Usecase{
		log:      log,
		tx:       tx,
		wallets:  wallets,
		payments: payments,
		provider: provider,
	}
}

// GetBalance возвращает баланс пользователя. Если кошелька нет — создаёт пустой.
func (u *Usecase) GetBalance(ctx context.Context, userID int64) (dto.WalletResponse, error) {
	u.log.DebugContext(ctx, "getting wallet balance",
		slog.String("op", opWalletGetBalance), slog.Int64("user_id", userID))

	w, err := u.wallets.GetOrCreate(ctx, userID)
	if err != nil {
		return dto.WalletResponse{}, fmt.Errorf("GetBalance: %w", err)
	}
	return dto.WalletResponse{Balance: w.Balance, Currency: currencyRUB}, nil
}

// ListTransactions возвращает ленту операций пользователя.
func (u *Usecase) ListTransactions(
	ctx context.Context, userID, cursor int64, limit int,
) (dto.WalletTransactionListResponse, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	u.log.DebugContext(ctx, "listing wallet transactions",
		slog.String("op", opWalletListTxs),
		slog.Int64("user_id", userID),
		slog.Int("limit", limit),
	)

	items, err := u.wallets.ListTransactionsByUser(ctx, userID, cursor, limit)
	if err != nil {
		return dto.WalletTransactionListResponse{}, fmt.Errorf("ListTransactions: %w", err)
	}

	resp := dto.WalletTransactionListResponse{Items: make([]dto.WalletTransactionResponse, 0, len(items))}
	for _, t := range items {
		resp.Items = append(resp.Items, dto.WalletTransactionResponse{
			ID:          t.ID,
			Amount:      t.Amount,
			Type:        t.Type,
			ReferenceID: t.ReferenceID,
			CreatedAt:   t.CreatedAt,
		})
	}
	if len(items) == limit {
		last := items[len(items)-1].ID
		resp.NextCursor = &last
	}
	return resp, nil
}

// Topup пополняет кошелёк. Идемпотентность по idempotency_key.
func (u *Usecase) Topup(
	ctx context.Context, userID, amount int64, idempotencyKey string,
) (dto.TopupWalletResponse, error) {
	if amount <= 0 || amount > maxTopupAmountRubles {
		return dto.TopupWalletResponse{}, ErrInvalidAmount
	}
	if idempotencyKey == "" {
		return dto.TopupWalletResponse{}, ErrInvalidRequest
	}

	u.log.InfoContext(ctx, "topup requested",
		slog.String("op", opWalletTopup),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
	)

	// Idempotency: если ключ уже использован — возвращаем тот же payment_id.
	if existing, err := u.wallets.GetTransactionByIdempotencyKey(ctx, idempotencyKey); err == nil {
		w, walErr := u.wallets.Get(ctx, userID)
		if walErr != nil {
			return dto.TopupWalletResponse{}, fmt.Errorf("Topup: idempotent path: %w", walErr)
		}
		paymentID := int64(0)
		if existing.ReferenceID != nil {
			paymentID = *existing.ReferenceID
		}
		return dto.TopupWalletResponse{Balance: w.Balance, PaymentID: paymentID}, nil
	}

	// Вызываем провайдера до открытия транзакции — он внешний и может быть медленным.
	// Mock мгновенный, но контракт сохраняем под реального провайдера.
	draft := models.Payment{UserID: userID, Amount: amount, Provider: models.PaymentProviderMock}
	status, providerRef, err := u.provider.InitPayment(ctx, draft)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: provider: %w", err)
	}
	if status != models.PaymentStatusSucceeded {
		return dto.TopupWalletResponse{}, ErrInvalidRequest
	}

	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ленивая инициализация кошелька + блокировка.
	if _, err = u.wallets.GetForUpdateTx(ctx, tx, userID); err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: lock wallet: %w", err)
	}

	refRef := providerRef
	payment, err := u.payments.InsertTx(
		ctx, tx, userID, amount, models.PaymentStatusSucceeded, models.PaymentProviderMock, &refRef,
	)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: insert payment: %w", err)
	}

	key := idempotencyKey
	if _, err = u.wallets.InsertTransactionTx(
		ctx, tx, userID, amount, models.WalletTxTypeTopup, &payment.ID, &key,
	); err != nil {
		// Гонка с параллельным запросом с тем же ключом — ловим повторно.
		if commercepostgres.IsPgUniqueViolation(err) {
			existing, getErr := u.wallets.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil {
				w, walErr := u.wallets.Get(ctx, userID)
				if walErr == nil {
					paymentID := int64(0)
					if existing.ReferenceID != nil {
						paymentID = *existing.ReferenceID
					}
					return dto.TopupWalletResponse{Balance: w.Balance, PaymentID: paymentID}, nil
				}
			}
		}
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: insert wallet_transaction: %w", err)
	}

	newBalance, err := u.wallets.IncrementBalanceTx(ctx, tx, userID, amount)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: increment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: commit: %w", err)
	}

	m := commercemetrics.Get()
	m.WalletTopups.Inc()
	m.WalletTopupsAmount.Add(float64(amount))

	u.log.InfoContext(ctx, "topup succeeded",
		slog.String("op", opWalletTopup),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
		slog.Int64("new_balance", newBalance),
	)

	return dto.TopupWalletResponse{Balance: newBalance, PaymentID: payment.ID}, nil
}
