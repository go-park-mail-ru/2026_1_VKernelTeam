package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

	cfg.MaxConns = 10
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres.New: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres.New: ping: %w", err)
	}

	return &Client{Pool: pool}, nil
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
