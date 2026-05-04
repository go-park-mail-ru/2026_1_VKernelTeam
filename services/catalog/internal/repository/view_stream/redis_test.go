package viewstream

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	"github.com/gomodule/redigo/redis"
)

func newStream(t *testing.T) (*ViewStream, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	pool := &redis.Pool{Dial: func() (redis.Conn, error) { return redis.Dial("tcp", mr.Addr()) }}
	s := New(pool, slog.New(slog.NewTextHandler(io.Discard, nil)), "stream:views", "view-cg", "worker-1")
	return s, mr
}

func TestEnsureConsumerGroup_FirstAndIdempotent(t *testing.T) {
	s, _ := newStream(t)
	if err := s.EnsureConsumerGroup(); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Повторный вызов не должен падать (BUSYGROUP-ошибка обрабатывается).
	if err := s.EnsureConsumerGroup(); err != nil {
		t.Fatalf("second: %v", err)
	}
}

func TestPublishAndReadBatch(t *testing.T) {
	s, _ := newStream(t)
	if err := s.EnsureConsumerGroup(); err != nil {
		t.Fatalf("group: %v", err)
	}

	uid := int64(42)
	err := s.Publish(context.Background(), models.ViewEvent{
		ProductID: 1, UserID: &uid, ViewedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	err = s.Publish(context.Background(), models.ViewEvent{
		ProductID: 2, UserID: nil, ViewedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("publish anonymous: %v", err)
	}

	events, err := s.ReadBatch(context.Background(), 10, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}

	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.MessageID
	}
	if err := s.Ack(context.Background(), ids); err != nil {
		t.Fatalf("ack: %v", err)
	}
}

func TestAck_EmptyIsNoOp(t *testing.T) {
	s, _ := newStream(t)
	if err := s.Ack(context.Background(), nil); err != nil {
		t.Fatalf("expected no-op: %v", err)
	}
}

func TestReadBatch_NoMessages(t *testing.T) {
	s, _ := newStream(t)
	if err := s.EnsureConsumerGroup(); err != nil {
		t.Fatalf("group: %v", err)
	}
	events, err := s.ReadBatch(context.Background(), 10, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("want 0 events, got %d", len(events))
	}
}
