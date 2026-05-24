package moderation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	tTrue  = "true"
	tFalse = "false"
)

type fakeStore struct {
	value    string
	getErr   error
	setErr   error
	gets     int
	setKey   string
	setValue string
	setBy    int64
}

func (f *fakeStore) Get(_ context.Context, key string) (string, error) {
	f.gets++
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.value, nil
}

func (f *fakeStore) Set(_ context.Context, key, value string, updatedBy int64) error {
	f.setKey = key
	f.setValue = value
	f.setBy = updatedBy
	return f.setErr
}

func newGate(store SettingStorage, ttl time.Duration) *Gate {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewGate(store, log, ttl)
}

func TestGate_IsEnabled_ReadsAndCaches(t *testing.T) {
	store := &fakeStore{value: tTrue}
	g := newGate(store, time.Minute)

	assert.True(t, g.IsEnabled(context.Background()))
	assert.True(t, g.IsEnabled(context.Background()))
	assert.Equal(t, 1, store.gets, "second call must hit cache")
}

func TestGate_IsEnabled_RefreshesAfterTTL(t *testing.T) {
	store := &fakeStore{value: tTrue}
	g := newGate(store, time.Nanosecond)

	assert.True(t, g.IsEnabled(context.Background()))
	time.Sleep(2 * time.Millisecond)
	store.value = tFalse
	assert.False(t, g.IsEnabled(context.Background()))
	assert.GreaterOrEqual(t, store.gets, 2)
}

func TestGate_IsEnabled_FalseOnReadError(t *testing.T) {
	store := &fakeStore{getErr: errors.New("db down")}
	g := newGate(store, time.Minute)
	assert.False(t, g.IsEnabled(context.Background()))
}

func TestGate_SetEnabled_PersistsAndUpdatesCache(t *testing.T) {
	store := &fakeStore{value: tFalse}
	g := newGate(store, time.Minute)

	assert.NoError(t, g.SetEnabled(context.Background(), true, 99))
	assert.Equal(t, "moderation_enabled", store.setKey)
	assert.Equal(t, tTrue, store.setValue)
	assert.Equal(t, int64(99), store.setBy)
	assert.True(t, g.IsEnabled(context.Background()))
}

func TestGate_SetEnabled_PropagatesError(t *testing.T) {
	store := &fakeStore{setErr: errors.New("boom")}
	g := newGate(store, time.Minute)
	assert.Error(t, g.SetEnabled(context.Background(), true, 1))
}

func TestGate_SetEnabled_FalseStored(t *testing.T) {
	store := &fakeStore{}
	g := newGate(store, time.Minute)
	assert.NoError(t, g.SetEnabled(context.Background(), false, 1))
	assert.Equal(t, tFalse, store.setValue)
}

func TestLoadSystemUserID_OK(t *testing.T) {
	id, err := LoadSystemUserID(context.Background(), &fakeStore{value: "123"})
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
}

func TestLoadSystemUserID_Missing(t *testing.T) {
	_, err := LoadSystemUserID(context.Background(), &fakeStore{getErr: errors.New("nope")})
	assert.ErrorIs(t, err, ErrSystemUserNotConfigured)
}

func TestLoadSystemUserID_BadValue(t *testing.T) {
	_, err := LoadSystemUserID(context.Background(), &fakeStore{value: "abc"})
	assert.Error(t, err)
}

func TestLoadSystemUserID_NonPositive(t *testing.T) {
	_, err := LoadSystemUserID(context.Background(), &fakeStore{value: "0"})
	assert.ErrorIs(t, err, ErrSystemUserNotConfigured)
}
