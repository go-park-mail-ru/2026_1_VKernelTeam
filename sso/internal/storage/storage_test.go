package storage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	path := "test_dump.json"
	defer os.Remove(path)

	st, err := New(path)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	uid, err := st.SaveUser(context.Background(), "foo@example.com", []byte("hash"))
	if err != nil {
		t.Fatalf("save user: %v", err)
	}

	if uid == 0 {
		t.Fatalf("expected non-zero uid")
	}

	// reopen storage from disk
	st2, err := New(path)
	if err != nil {
		t.Fatalf("failed to reload storage: %v", err)
	}

	u, err := st2.User(context.Background(), "foo@example.com")
	if err != nil {
		t.Fatalf("retrieve user after reload: %v", err)
	}
	if u.ID != uid {
		t.Fatalf("id mismatch: got %d want %d", u.ID, uid)
	}
}

func TestConcurrency(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", i)
			if _, err := st.SaveUser(context.Background(), email, []byte("h")); err != nil {
				t.Errorf("save %s: %v", email, err)
			}
		}(i)
	}
	wg.Wait()

	// verify all users exist
	for i := 0; i < n; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		if _, err := st.User(context.Background(), email); err != nil {
			t.Errorf("missing user %s: %v", email, err)
		}
	}
}
