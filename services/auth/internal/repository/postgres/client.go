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

// Параметры пула.
//
// MaxConns = 10 на реплику. Auth — самый «лёгкий» по запросам сервис
// (login/register/refresh). В типовом сценарии активных запросов в БД
// одновременно — 3-4, остальные 6-7 нужны как buffer на пик.
//
// MinConns = 2 — держим тёплыми, чтобы первое попадание после простоя
// не дожидалось TCP-handshake + SCRAM (это ~30-100мс на холодный коннект).
//
// max_connections на сервере — 150 (см. db/config/postgresql.custom.conf).
// При 3 репликах auth даём максимум 30 коннектов, что укладывается в
// CONNECTION LIMIT 25 на роль с запасом на горячую миграцию (роль
// держит лимит даже при flapping пода).
const (
	maxConns         = 10
	minConns         = 2
	maxConnLifetime  = 30 * time.Minute
	maxConnIdleTime  = 5 * time.Minute
	healthCheckEvery = 30 * time.Second
)

// Клиентские таймауты. Дублируют серверные (см. postgresql.custom.conf),
// но задаются именно через RuntimeParams — это надёжнее, чем options=
// в DSN: pgx прописывает их в каждое новое соединение перед выдачей в пул.
//
// Обоснование значений: см. db/README.md, раздел «Таймауты».
const (
	statementTimeoutMS               = "30000" // 30 секунд
	lockTimeoutMS                    = "5000"  // 5 секунд
	idleInTxSessionTimeoutMS         = "60000" // 60 секунд
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

	// Клиентские GUC применяются ко всем новым backend-сессиям в пуле.
	// Это нужно, чтобы серверные дефолты из postgresql.custom.conf
	// не были незаметно переопределены каким-нибудь default'ом драйвера.
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

// applyClientTimeouts проставляет statement_timeout, lock_timeout и
// idle_in_transaction_session_timeout как RuntimeParams ConnConfig.
// pgx отправит SET ... при установке каждого нового соединения.
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
