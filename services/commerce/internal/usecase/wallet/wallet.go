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
	paymentuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/payment"
)

const (
	opWalletTopup            = "usecase.wallet.Topup"
	opWalletApplyProviderUpd = "usecase.wallet.ApplyProviderUpdate"
	opWalletGetBalance       = "usecase.wallet.GetBalance"
	opWalletListTxs          = "usecase.wallet.ListTransactions"
	currencyRUB              = "RUB"
	defaultListLimit         = 20
	maxListLimit             = 100
	maxTopupAmountRubles     = 1_000_000
)

// Бизнес-ошибки usecase.
var (
	ErrInvalidAmount   = errors.New("INVALID_AMOUNT")
	ErrInvalidRequest  = errors.New("INVALID_REQUEST")
	ErrAmountTooLarge  = errors.New("INVALID_AMOUNT")
	ErrPaymentNotFound = errors.New("PAYMENT_NOT_FOUND")
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
	GetByID(ctx context.Context, id int64) (models.Payment, error)
	GetByProviderRefForUpdateTx(ctx context.Context, tx pgx.Tx, provider, providerRef string) (models.Payment, error)
	UpdateProviderRefTx(ctx context.Context, tx pgx.Tx, id int64, providerRef, confirmationURL string, rawPayload []byte) error
	UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int64, status string, rawPayload []byte) error
}

// PaymentProvider — внешний платёжный сервис (см. usecase/payment).
type PaymentProvider interface {
	InitPayment(ctx context.Context, p models.Payment, idempotencyKey string) (paymentuc.InitResult, error)
	GetPayment(ctx context.Context, providerRef string) (paymentuc.InitResult, error)
}

// Usecase реализует Topup, ApplyProviderUpdate и чтение кошелька.
type Usecase struct {
	log          *slog.Logger
	tx           TxRunner
	wallets      WalletRepo
	payments     PaymentRepo
	provider     PaymentProvider
	providerName string // models.PaymentProviderMock | models.PaymentProviderYooKassa
}

// New создаёт usecase кошелька с зависимостями репозиториев и провайдера платежей.
// providerName — идентификатор провайдера, который будет записываться в payment.provider.
func New(
	log *slog.Logger,
	tx TxRunner,
	wallets WalletRepo,
	payments PaymentRepo,
	provider PaymentProvider,
	providerName string,
) *Usecase {
	return &Usecase{
		log:          log,
		tx:           tx,
		wallets:      wallets,
		payments:     payments,
		provider:     provider,
		providerName: providerName,
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

// Topup инициирует пополнение через провайдера. Возвращает payment_id, статус и
// (для асинхронных провайдеров) confirmation_url для редиректа пользователя.
//
// Поведение:
//   - mock-провайдер возвращает status='succeeded' сразу — баланс зачисляется
//     в этом же вызове;
//   - ЮКасса возвращает status='pending' — payment создаётся, баланс НЕ
//     меняется, фронт делает редирект на confirmation_url. Зачисление
//     произойдёт через webhook или reconciler.
//
// Идемпотентность по idempotency_key:
//   - если ключ уже использовался для succeeded — возвращаем тот же payment_id
//     и текущий баланс;
//   - если ключ уже использовался для pending — возвращаем тот же payment_id
//     и confirmation_url.
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

	// Idempotency: ключ уже сожжён → платёж succeeded, возвращаем баланс и payment_id.
	if existing, err := u.wallets.GetTransactionByIdempotencyKey(ctx, idempotencyKey); err == nil {
		w, walErr := u.wallets.Get(ctx, userID)
		if walErr != nil {
			return dto.TopupWalletResponse{}, fmt.Errorf("Topup: idempotent path: %w", walErr)
		}
		paymentID := int64(0)
		if existing.ReferenceID != nil {
			paymentID = *existing.ReferenceID
		}
		return dto.TopupWalletResponse{Balance: w.Balance, PaymentID: paymentID, Status: models.PaymentStatusSucceeded}, nil
	}

	// 1. Создаём pending-платёж (без provider_ref — он появится после ответа провайдера).
	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = u.wallets.GetForUpdateTx(ctx, tx, userID); err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: ensure wallet: %w", err)
	}

	pending, err := u.payments.InsertTx(
		ctx, tx, userID, amount, models.PaymentStatusPending, u.providerName, nil,
	)
	if err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: insert pending payment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: commit pending: %w", err)
	}

	// 2. Вызываем провайдера ВНЕ транзакции (внешний вызов, может быть медленным).
	draft := models.Payment{ID: pending.ID, UserID: userID, Amount: amount, Provider: u.providerName}
	res, err := u.provider.InitPayment(ctx, draft, idempotencyKey)
	if err != nil {
		// Помечаем платёж failed, баланс не трогаем.
		u.markFailed(ctx, pending.ID, nil)
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: provider: %w", err)
	}

	// 3. Сохраняем provider_ref + confirmation_url у платежа.
	if err := u.attachProviderRef(ctx, pending.ID, res); err != nil {
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: attach provider ref: %w", err)
	}

	// 4. Если провайдер вернул терминальный статус сразу (mock или авто-capture) —
	// сжигаем idempotency_key и зачисляем баланс.
	switch res.Status {
	case models.PaymentStatusSucceeded:
		balance, err := u.applySucceeded(ctx, pending.ID, userID, amount, idempotencyKey, res.RawPayload)
		if err != nil {
			return dto.TopupWalletResponse{}, fmt.Errorf("Topup: apply succeeded: %w", err)
		}
		return dto.TopupWalletResponse{
			Balance:   balance,
			PaymentID: pending.ID,
			Status:    models.PaymentStatusSucceeded,
		}, nil
	case models.PaymentStatusFailed, models.PaymentStatusCancelled:
		u.markFailed(ctx, pending.ID, res.RawPayload)
		return dto.TopupWalletResponse{}, ErrInvalidRequest
	case models.PaymentStatusPending:
		return dto.TopupWalletResponse{
			PaymentID:       pending.ID,
			Status:          models.PaymentStatusPending,
			ConfirmationURL: res.ConfirmationURL,
		}, nil
	default:
		return dto.TopupWalletResponse{}, fmt.Errorf("Topup: unexpected provider status %q", res.Status)
	}
}

// ApplyProviderUpdate применяет терминальный статус платежа (succeeded/failed/cancelled),
// пришедший из webhook'а или reconciler'а. Идемпотентен: повторный вызов на уже
// применённом платеже возвращает текущее состояние без побочных эффектов.
//
// idempotencyKey — ключ для wallet_transaction; для webhook-применения генерируется
// детерминированно из payment_id (см. helpers ниже).
func (u *Usecase) ApplyProviderUpdate(
	ctx context.Context,
	provider, providerRef, status string,
	rawPayload []byte,
) error {
	u.log.InfoContext(ctx, "applying provider update",
		slog.String("op", opWalletApplyProviderUpd),
		slog.String("provider", provider),
		slog.String("status", status),
	)

	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ApplyProviderUpdate: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p, err := u.payments.GetByProviderRefForUpdateTx(ctx, tx, provider, providerRef)
	if err != nil {
		return fmt.Errorf("ApplyProviderUpdate: lookup: %w", err)
	}

	// Уже применён — выходим без побочных эффектов.
	if p.Status != models.PaymentStatusPending {
		return nil
	}

	switch status {
	case models.PaymentStatusSucceeded:
		if _, err := u.wallets.GetForUpdateTx(ctx, tx, p.UserID); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: lock wallet: %w", err)
		}

		idemKey := webhookIdempotencyKey(p.ID)
		refID := p.ID
		if _, err := u.wallets.InsertTransactionTx(
			ctx, tx, p.UserID, p.Amount, models.WalletTxTypeTopup, &refID, &idemKey,
		); err != nil {
			// Параллельный вебхук/reconciler — повторим выход без изменений.
			if commercepostgres.IsPgUniqueViolation(err) {
				return nil
			}
			return fmt.Errorf("ApplyProviderUpdate: insert wallet_tx: %w", err)
		}

		if _, err := u.wallets.IncrementBalanceTx(ctx, tx, p.UserID, p.Amount); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: increment: %w", err)
		}

		if err := u.payments.UpdateStatusTx(ctx, tx, p.ID, models.PaymentStatusSucceeded, rawPayload); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: update payment: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: commit: %w", err)
		}

		m := commercemetrics.Get()
		m.WalletTopups.Inc()
		m.WalletTopupsAmount.Add(float64(p.Amount))
		u.log.InfoContext(ctx, "topup applied via provider update",
			slog.Int64("payment_id", p.ID),
			slog.Int64("user_id", p.UserID),
			slog.Int64("amount", p.Amount),
		)
		return nil

	case models.PaymentStatusFailed, models.PaymentStatusCancelled:
		if err := u.payments.UpdateStatusTx(ctx, tx, p.ID, status, rawPayload); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: update payment: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("ApplyProviderUpdate: commit failed status: %w", err)
		}
		return nil

	default:
		// pending/unknown — игнорируем (webhook должен слать только терминальные).
		return nil
	}
}

// GetPaymentStatus отдаёт текущий статус платежа (для фронта после return_url).
func (u *Usecase) GetPaymentStatus(ctx context.Context, userID, paymentID int64) (dto.PaymentStatusResponse, error) {
	p, err := u.payments.GetByID(ctx, paymentID)
	if err != nil {
		return dto.PaymentStatusResponse{}, fmt.Errorf("GetPaymentStatus: %w", err)
	}
	if p.UserID != userID {
		return dto.PaymentStatusResponse{}, ErrPaymentNotFound
	}
	resp := dto.PaymentStatusResponse{
		PaymentID: p.ID,
		Status:    p.Status,
		Amount:    p.Amount,
	}
	if p.ConfirmationURL != nil {
		resp.ConfirmationURL = *p.ConfirmationURL
	}
	return resp, nil
}

// attachProviderRef сохраняет provider_ref + confirmation_url в отдельной транзакции.
func (u *Usecase) attachProviderRef(ctx context.Context, paymentID int64, res paymentuc.InitResult) error {
	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("attachProviderRef: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := u.payments.UpdateProviderRefTx(
		ctx, tx, paymentID, res.ProviderRef, res.ConfirmationURL, res.RawPayload,
	); err != nil {
		return fmt.Errorf("attachProviderRef: update: %w", err)
	}
	return tx.Commit(ctx)
}

// applySucceeded сжигает idempotencyKey пользователя и зачисляет баланс
// (для синхронных провайдеров типа mock).
func (u *Usecase) applySucceeded(
	ctx context.Context,
	paymentID, userID, amount int64,
	idempotencyKey string,
	rawPayload []byte,
) (int64, error) {
	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("applySucceeded: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := u.wallets.GetForUpdateTx(ctx, tx, userID); err != nil {
		return 0, fmt.Errorf("applySucceeded: lock wallet: %w", err)
	}

	key := idempotencyKey
	refID := paymentID
	if _, err := u.wallets.InsertTransactionTx(
		ctx, tx, userID, amount, models.WalletTxTypeTopup, &refID, &key,
	); err != nil {
		if commercepostgres.IsPgUniqueViolation(err) {
			// Гонка по idempotency_key — выходим, баланс уже зачислен параллельным запросом.
			w, getErr := u.wallets.Get(ctx, userID)
			if getErr == nil {
				return w.Balance, nil
			}
		}
		return 0, fmt.Errorf("applySucceeded: insert wallet_tx: %w", err)
	}

	newBalance, err := u.wallets.IncrementBalanceTx(ctx, tx, userID, amount)
	if err != nil {
		return 0, fmt.Errorf("applySucceeded: increment: %w", err)
	}

	if err := u.payments.UpdateStatusTx(ctx, tx, paymentID, models.PaymentStatusSucceeded, rawPayload); err != nil {
		return 0, fmt.Errorf("applySucceeded: update payment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("applySucceeded: commit: %w", err)
	}

	m := commercemetrics.Get()
	m.WalletTopups.Inc()
	m.WalletTopupsAmount.Add(float64(amount))
	return newBalance, nil
}

// markFailed переводит платёж в failed без зачисления баланса. Ошибки логируются и проглатываются.
func (u *Usecase) markFailed(ctx context.Context, paymentID int64, rawPayload []byte) {
	tx, err := u.tx.Begin(ctx)
	if err != nil {
		u.log.ErrorContext(ctx, "markFailed: begin", slog.String("error", err.Error()))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := u.payments.UpdateStatusTx(ctx, tx, paymentID, models.PaymentStatusFailed, rawPayload); err != nil {
		u.log.ErrorContext(ctx, "markFailed: update", slog.String("error", err.Error()))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		u.log.ErrorContext(ctx, "markFailed: commit", slog.String("error", err.Error()))
	}
}

// webhookIdempotencyKey генерирует детерминированный ключ для wallet_transaction,
// чтобы повторные применения одного и того же payment'а не задвоили зачисление.
func webhookIdempotencyKey(paymentID int64) string {
	return fmt.Sprintf("payment-apply-%d", paymentID)
}
