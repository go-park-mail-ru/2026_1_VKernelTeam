package blacklist

import (
	"testing"
	"time"
)

func TestInMemory_AddAndCheck(t *testing.T) {
	storage := New(5 * time.Minute)

	// Добавляем токен
	storage.Add("test-jti", time.Now().Add(time.Hour))

	// Проверяем, что он есть
	if !storage.Check("test-jti") {
		t.Errorf("expected token to be in blacklist")
	}

	// Проверяем несуществующий токен
	if storage.Check("non-existent") {
		t.Errorf("expected token not to be in blacklist")
	}
}

func TestInMemory_Sweeper(t *testing.T) {
	// Создаем хранилище с коротким интервалом для теста
	storage := New(150 * time.Millisecond)

	// Добавляем токен с истекшим сроком
	storage.Add("expired-jti", time.Now().Add(-time.Hour))

	// Даем время на работу sweeper
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что токен удален
	if storage.Check("expired-jti") {
		t.Errorf("expected expired token to be removed by sweeper")
	}
}
