package blacklist

import (
	"context"
	"sync"
	"time"
)

// InMemory реализует хранилище отозванных токенов
type InMemory struct {
	mu              sync.RWMutex
	tokens          map[string]time.Time // key: jti, value: expiration time
	ctx             context.Context
	cancel          context.CancelFunc
	cleanupInterval time.Duration
	sweeperDone     chan struct{}
}

func New(cleanupInterval time.Duration) *InMemory {
	ctx, cancel := context.WithCancel(context.Background())

	blacklist := &InMemory{
		tokens:          make(map[string]time.Time),
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: cleanupInterval,
		sweeperDone:     make(chan struct{}),
	}
	// Запускаем горутину для очистки чёрного списка
	go blacklist.startSweeper()

	return blacklist
}

// Add добавляет токен в блэк-лист
func (s *InMemory) Add(jti string, exp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[jti] = exp
}

// Check проверяет наличие токена в чёрном списке
func (s *InMemory) Check(jti string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.tokens[jti]
	return exists
}

// startSweeper подчищает мапу, предотвращая утечку памяти
func (s *InMemory) startSweeper() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer func() {
		ticker.Stop()
		close(s.sweeperDone)
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup удаляет истёкшие токены из чёрного списка
func (s *InMemory) cleanup() {
	// Получаем список ключей под блокировкой (быстро)
	s.mu.Lock()
	keysToDelete := make([]string, 0)
	now := time.Now()
	for jti, exp := range s.tokens {
		if now.After(exp) {
			keysToDelete = append(keysToDelete, jti)
		}
	}
	// Удаляем в той же блокировке, но список уже подготовлен
	for _, jti := range keysToDelete {
		delete(s.tokens, jti)
	}
	s.mu.Unlock()
}

// Stop корректно завершает горутину sweeper'а (graceful shutdown)
func (s *InMemory) Stop() {
	s.cancel()
	<-s.sweeperDone
}
