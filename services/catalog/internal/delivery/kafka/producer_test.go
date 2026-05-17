package kafka

import (
	"io"
	"log/slog"
	"testing"
)

// Producer/Close — без реального брокера. Полные тесты на Publish*
// требуют поднятого Kafka (или mock writer'а через интерфейс — задача на
// последующий рефакторинг).

func TestNewProducer_NotNil(t *testing.T) {
	p := NewProducer([]string{testBrokerAddr}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if p == nil {
		t.Fatal("expected producer instance")
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
