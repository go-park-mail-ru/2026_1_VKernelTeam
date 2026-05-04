package redis

import "testing"

func TestNewAndClose(t *testing.T) {
	c := New("localhost:6379")

	if c == nil {
		t.Fatal("expected cache instance")
	}

	if c.Pool() == nil {
		t.Fatal("expected non-nil pool")
	}

	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
