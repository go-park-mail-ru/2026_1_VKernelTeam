package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// Storage описывает хранилище для блэклиста и refresh-токенов в Redis
type Storage struct {
	pool *redis.Pool
}

// New создаёт новый экземпляр Storage с указанным адресом Redis
func New(addr string) *Storage {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
	return &Storage{pool: pool}
}

// Close закрывает пул соединений с Redis
func (s *Storage) Close() error {
	return s.pool.Close()
}

// --- TokenRevoker ---

// Add добавляет JTI в блэклист с указанным временем жизни
func (s *Storage) Add(jti string, exp time.Time) {
	conn := s.pool.Get()
	defer conn.Close()

	ttl := time.Until(exp)
	if ttl <= 0 {
		return
	}

	key := fmt.Sprintf("blacklist:%s", jti)
	// Сохраняем "1" с TTL
	_, _ = conn.Do("SET", key, "1", "PX", int64(ttl/time.Millisecond))
}

// Check проверяет, находится ли JTI в блэклисте
func (s *Storage) Check(jti string) bool {
	conn := s.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("blacklist:%s", jti)
	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	return exists
}

// --- RefreshStorage ---

// SaveRefresh сохраняет refresh-токен в Redis с указанным TTL
func (s *Storage) SaveRefresh(ctx context.Context, token string, userID int64, ttl time.Duration) error {
	conn := s.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("refresh:%s", token)
	_, err := conn.Do("SET", key, userID, "PX", int64(ttl/time.Millisecond))
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

// GetRefresh получает ID пользователя по refresh-токену
func (s *Storage) GetRefresh(ctx context.Context, token string) (int64, error) {
	conn := s.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("refresh:%s", token)
	userID, err := redis.Int64(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return 0, fmt.Errorf("refresh token not found")
		}
		return 0, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return userID, nil
}

// DeleteRefresh удаляет refresh-токен из Redis
func (s *Storage) DeleteRefresh(ctx context.Context, token string) error {
	conn := s.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("refresh:%s", token)
	_, err := conn.Do("DEL", key)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}
