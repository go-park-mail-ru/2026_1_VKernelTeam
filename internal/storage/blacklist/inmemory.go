package blacklist

import (
	"sync"
	"time"
)

// InMemory реализует хранилище отозванных токенов
type InMemory struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // key: jti, value: expiration time
}

func New(cleanupInterval time.Duration) *InMemory {
	blacklist := &InMemory{
		tokens: make(map[string]time.Time),
	}
	// Запускаем горутину для очистки чёрного списка
	go blacklist.startSweeper(cleanupInterval)

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
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.tokens[jti]
	return exists
}

// startSweeper подчищает мапу, предотвращая утечку памяти
func (s *InMemory) startSweeper(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for range ticker.C {
		s.mu.Lock()
		for jti, exp := range s.tokens {
			if time.Now().After(exp) {
				delete(s.tokens, jti)
			}
		}
		s.mu.Unlock()
	}
}
