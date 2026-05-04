package viewcache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
)

func newCache(t *testing.T) (*ViewCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	pool := &redis.Pool{Dial: func() (redis.Conn, error) { return redis.Dial("tcp", mr.Addr()) }}
	return New(pool, time.Hour, 24*time.Hour), mr
}

func TestCheckAndSetDedup_FirstAndDuplicate(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	isNew, err := c.CheckAndSetDedup(ctx, 1, "device-A")
	if err != nil || !isNew {
		t.Fatalf("first call: isNew=%v err=%v", isNew, err)
	}
	isNew, err = c.CheckAndSetDedup(ctx, 1, "device-A")
	if err != nil || isNew {
		t.Fatalf("dup call: isNew=%v err=%v", isNew, err)
	}
}

func TestIncrementAndGetCount(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	count, err := c.IncrementCount(ctx, 5)
	if err != nil || count != 1 {
		t.Fatalf("first incr: %d %v", count, err)
	}
	count, err = c.IncrementCount(ctx, 5)
	if err != nil || count != 2 {
		t.Fatalf("second incr: %d %v", count, err)
	}

	got, hit, err := c.GetCount(ctx, 5)
	if err != nil || !hit || got != 2 {
		t.Fatalf("get: %d hit=%v err=%v", got, hit, err)
	}
}

func TestGetCount_Miss(t *testing.T) {
	c, _ := newCache(t)
	got, hit, err := c.GetCount(context.Background(), 999)
	if err != nil || hit || got != 0 {
		t.Fatalf("miss expected: %d hit=%v err=%v", got, hit, err)
	}
}

func TestSetCount(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()
	if err := c.SetCount(ctx, 7, 100); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, hit, err := c.GetCount(ctx, 7)
	if err != nil || !hit || got != 100 {
		t.Fatalf("get after set: %d hit=%v err=%v", got, hit, err)
	}
}
