// Package wallet — хранилище кошельков и операций.
package wallet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opWalletGet               = "db.wallet.Get"
	opWalletGetOrCreate       = "db.wallet.GetOrCreate"
	opWalletGetForUpdate      = "db.wallet.GetForUpdate"
	opWalletIncrement         = "db.wallet.Increment"
	opWalletDecrement         = "db.wallet.Decrement"
	opWalletTxInsert          = "db.wallet.TxInsert"
	opWalletTxListByUser      = "db.wallet.TxListByUser"
	opWalletTxGetByIdempotKey = "db.wallet.TxGetByIdempotencyKey"
)

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrTransactionNotFound = errors.New("wallet transaction not found")
)

// PgxPoolTx — пул с поддержкой транзакций.
type PgxPoolTx interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// WalletStorage отвечает за кошельки и лог транзакций.
type WalletStorage struct {
	pool PgxPoolTx
	log  *slog.Logger
}

func NewWalletStorage(pool PgxPoolTx, log *slog.Logger) *WalletStorage {
	return &WalletStorage{pool: pool, log: log}
}

// Pool возвращает пул для усcase'а, чтобы тот мог открыть транзакцию.
func (s *WalletStorage) Pool() PgxPoolTx { return s.pool }

// Get возвращает кошелёк пользователя. Если записи нет — ErrWalletNotFound.
func (s *WalletStorage) Get(ctx context.Context, userID int64) (models.Wallet, error) {
	const query = `SELECT user_id, balance, updated_at FROM wallet WHERE user_id = $1`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletGet),
		slog.Int64("user_id", userID),
	)

	var w models.Wallet
	err := s.pool.QueryRow(ctx, query, userID).Scan(&w.UserID, &w.Balance, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Wallet{}, ErrWalletNotFound
		}
		return models.Wallet{}, fmt.Errorf("WalletStorage.Get: %w", err)
	}
	return w, nil
}

// GetOrCreate возвращает существующий кошелёк или создаёт новый с нулевым балансом.
func (s *WalletStorage) GetOrCreate(ctx context.Context, userID int64) (models.Wallet, error) {
	const query = `
		INSERT INTO wallet (user_id, balance)
		VALUES ($1, 0)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = wallet.updated_at
		RETURNING user_id, balance, updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletGetOrCreate),
		slog.Int64("user_id", userID),
	)

	var w models.Wallet
	err := s.pool.QueryRow(ctx, query, userID).Scan(&w.UserID, &w.Balance, &w.UpdatedAt)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("WalletStorage.GetOrCreate: %w", err)
	}
	return w, nil
}

// GetForUpdateTx внутри транзакции выбирает кошелёк с блокировкой строки.
// Если записи нет — создаёт с балансом 0 и возвращает с теми же гарантиями блокировки.
func (s *WalletStorage) GetForUpdateTx(ctx context.Context, tx pgx.Tx, userID int64) (models.Wallet, error) {
	const ensureQuery = `
		INSERT INTO wallet (user_id, balance) VALUES ($1, 0)
		ON CONFLICT (user_id) DO NOTHING
	`
	if _, err := tx.Exec(ctx, ensureQuery, userID); err != nil {
		return models.Wallet{}, fmt.Errorf("WalletStorage.GetForUpdateTx: ensure: %w", err)
	}

	const selectQuery = `
		SELECT user_id, balance, updated_at
		FROM wallet
		WHERE user_id = $1
		FOR UPDATE
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletGetForUpdate),
		slog.Int64("user_id", userID),
	)

	var w models.Wallet
	err := tx.QueryRow(ctx, selectQuery, userID).Scan(&w.UserID, &w.Balance, &w.UpdatedAt)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("WalletStorage.GetForUpdateTx: select: %w", err)
	}
	return w, nil
}

// IncrementBalanceTx добавляет к балансу указанную сумму внутри транзакции.
func (s *WalletStorage) IncrementBalanceTx(ctx context.Context, tx pgx.Tx, userID, amount int64) (int64, error) {
	const query = `
		UPDATE wallet
		SET balance = balance + $2, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		RETURNING balance
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletIncrement),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
	)

	var newBalance int64
	if err := tx.QueryRow(ctx, query, userID, amount).Scan(&newBalance); err != nil {
		return 0, fmt.Errorf("WalletStorage.IncrementBalanceTx: %w", err)
	}
	return newBalance, nil
}

// DecrementBalanceTx списывает с баланса указанную сумму. CHECK balance >= 0 защитит от ухода в минус.
func (s *WalletStorage) DecrementBalanceTx(ctx context.Context, tx pgx.Tx, userID, amount int64) (int64, error) {
	const query = `
		UPDATE wallet
		SET balance = balance - $2, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		RETURNING balance
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletDecrement),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
	)

	var newBalance int64
	if err := tx.QueryRow(ctx, query, userID, amount).Scan(&newBalance); err != nil {
		return 0, fmt.Errorf("WalletStorage.DecrementBalanceTx: %w", err)
	}
	return newBalance, nil
}

// InsertTransactionTx записывает операцию в лог. Возвращает saved-объект.
// При нарушении UNIQUE(idempotency_key) поднимется pg-ошибка — обработка на usecase'е.
func (s *WalletStorage) InsertTransactionTx(
	ctx context.Context,
	tx pgx.Tx,
	userID, amount int64,
	txType string,
	referenceID *int64,
	idempotencyKey *string,
) (models.WalletTransaction, error) {
	const query = `
		INSERT INTO wallet_transaction (user_id, amount, type, reference_id, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, amount, type, reference_id, idempotency_key, created_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletTxInsert),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
		slog.String("type", txType),
	)

	var t models.WalletTransaction
	err := tx.QueryRow(ctx, query, userID, amount, txType, referenceID, idempotencyKey).Scan(
		&t.ID, &t.UserID, &t.Amount, &t.Type, &t.ReferenceID, &t.IdempotencyKey, &t.CreatedAt,
	)
	if err != nil {
		return models.WalletTransaction{}, fmt.Errorf("WalletStorage.InsertTransactionTx: %w", err)
	}
	return t, nil
}

// GetTransactionByIdempotencyKey ищет операцию по ключу идемпотентности.
func (s *WalletStorage) GetTransactionByIdempotencyKey(
	ctx context.Context,
	key string,
) (models.WalletTransaction, error) {
	const query = `
		SELECT id, user_id, amount, type, reference_id, idempotency_key, created_at
		FROM wallet_transaction
		WHERE idempotency_key = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletTxGetByIdempotKey),
	)

	var t models.WalletTransaction
	err := s.pool.QueryRow(ctx, query, key).Scan(
		&t.ID, &t.UserID, &t.Amount, &t.Type, &t.ReferenceID, &t.IdempotencyKey, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WalletTransaction{}, ErrTransactionNotFound
		}
		return models.WalletTransaction{}, fmt.Errorf("WalletStorage.GetTransactionByIdempotencyKey: %w", err)
	}
	return t, nil
}

// ListTransactionsByUser возвращает ленту операций пользователя с курсорной пагинацией.
func (s *WalletStorage) ListTransactionsByUser(
	ctx context.Context,
	userID int64,
	cursor int64,
	limit int,
) ([]models.WalletTransaction, error) {
	const query = `
		SELECT id, user_id, amount, type, reference_id, idempotency_key, created_at
		FROM wallet_transaction
		WHERE user_id = $1 AND ($2 = 0 OR id < $2)
		ORDER BY id DESC
		LIMIT $3
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opWalletTxListByUser),
		slog.Int64("user_id", userID),
	)

	rows, err := s.pool.Query(ctx, query, userID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("WalletStorage.ListTransactionsByUser: query: %w", err)
	}
	defer rows.Close()

	var items []models.WalletTransaction
	for rows.Next() {
		var t models.WalletTransaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Amount, &t.Type,
			&t.ReferenceID, &t.IdempotencyKey, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WalletStorage.ListTransactionsByUser: scan: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WalletStorage.ListTransactionsByUser: rows: %w", err)
	}
	if items == nil {
		items = []models.WalletTransaction{}
	}
	return items, nil
}
