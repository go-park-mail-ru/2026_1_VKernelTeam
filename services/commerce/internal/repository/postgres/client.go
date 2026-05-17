package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Параметры пула — обоснование описано в db/README.md.
// Commerce работает с транзакциями (оформление заказа, чаты),
// нужен запас на параллельные tx без long-tail latency.
const (
	maxConns         = 12
	minConns         = 3
	maxConnLifetime  = 30 * time.Minute
	maxConnIdleTime  = 5 * time.Minute
	healthCheckEvery = 30 * time.Second
)

const (
	statementTimeoutMS       = "30000"
	lockTimeoutMS            = "5000"
	idleInTxSessionTimeoutMS = "60000"
)

// Client реализует обертку над пулом соединений к PostgreSQL.
type Client struct {
	Pool *pgxpool.Pool
}

// New создаёт пул соединений и проверяет доступность БД.
func New(dsn string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.New: parse config: %w", err)
	}

	applyClientTimeouts(cfg.ConnConfig)

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.MaxConnIdleTime = maxConnIdleTime
	cfg.HealthCheckPeriod = healthCheckEvery

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres.New: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres.New: ping: %w", err)
	}

	return &Client{Pool: pool}, nil
}

func applyClientTimeouts(cc *pgx.ConnConfig) {
	if cc.RuntimeParams == nil {
		cc.RuntimeParams = make(map[string]string)
	}
	cc.RuntimeParams["statement_timeout"] = statementTimeoutMS
	cc.RuntimeParams["lock_timeout"] = lockTimeoutMS
	cc.RuntimeParams["idle_in_transaction_session_timeout"] = idleInTxSessionTimeoutMS
}

// Close закрывает пул соединений.
func (c *Client) Close() {
	c.Pool.Close()
}

// IsPgUniqueViolation проверяет, является ли ошибка нарушением UNIQUE в PostgreSQL.
func IsPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
