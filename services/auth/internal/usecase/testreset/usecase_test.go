package testreset

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
)

type stubLookup struct {
	user models.User
	err  error
}

func (s stubLookup) UserByID(_ context.Context, _ int64) (models.User, error) {
	return s.user, s.err
}

type stubWiper struct {
	called bool
	err    error
}

func (s *stubWiper) WipeUserData(_ context.Context, _ int64) error {
	s.called = true
	return s.err
}

func nullLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestReset_TestUser_CallsWipe(t *testing.T) {
	wiper := &stubWiper{}
	uc := New(
		nullLogger(),
		stubLookup{user: models.User{ID: 42, Email: "clover-tester1@clover.ru"}},
		wiper,
	)

	if err := uc.Reset(context.Background(), 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wiper.called {
		t.Fatal("expected WipeUserData to be called")
	}
}

func TestReset_TestUser_PrefixCaseInsensitive(t *testing.T) {
	wiper := &stubWiper{}
	uc := New(
		nullLogger(),
		stubLookup{user: models.User{ID: 7, Email: "CLOVER-TESTER+alt@clover.ru"}},
		wiper,
	)

	if err := uc.Reset(context.Background(), 7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wiper.called {
		t.Fatal("expected WipeUserData to be called for upper-case prefix")
	}
}

func TestReset_NotTestUser_ReturnsErrAndSkipsWipe(t *testing.T) {
	wiper := &stubWiper{}
	uc := New(
		nullLogger(),
		stubLookup{user: models.User{ID: 99, Email: "alice@clover.ru"}},
		wiper,
	)

	err := uc.Reset(context.Background(), 99)
	if !errors.Is(err, ErrNotTestUser) {
		t.Fatalf("expected ErrNotTestUser, got %v", err)
	}
	if wiper.called {
		t.Fatal("WipeUserData must not be called for non-test users")
	}
}

func TestReset_LookupError_PropagatesAndSkipsWipe(t *testing.T) {
	wantErr := errors.New("db down")
	wiper := &stubWiper{}
	uc := New(
		nullLogger(),
		stubLookup{err: wantErr},
		wiper,
	)

	err := uc.Reset(context.Background(), 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected lookup error to propagate, got %v", err)
	}
	if wiper.called {
		t.Fatal("WipeUserData must not be called when lookup fails")
	}
}

func TestReset_WipeError_Propagates(t *testing.T) {
	wantErr := errors.New("delete failed")
	wiper := &stubWiper{err: wantErr}
	uc := New(
		nullLogger(),
		stubLookup{user: models.User{ID: 5, Email: "clover-tester+e2e@clover.ru"}},
		wiper,
	)

	err := uc.Reset(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wipe error to propagate, got %v", err)
	}
	if !wiper.called {
		t.Fatal("wiper should have been invoked")
	}
}
