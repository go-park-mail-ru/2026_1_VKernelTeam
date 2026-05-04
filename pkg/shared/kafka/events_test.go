package kafka

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_OK(t *testing.T) {
	payload := UserPayload{UserID: 42}
	e, err := NewEvent(EventUserUpdated, payload)
	require.NoError(t, err)

	assert.Equal(t, EventUserUpdated, e.EventType)
	assert.NotEmpty(t, e.EventID)
	assert.False(t, e.Timestamp.IsZero())

	var got UserPayload
	require.NoError(t, json.Unmarshal(e.Payload, &got))
	assert.Equal(t, payload, got)
}

func TestNewEvent_MarshalError(t *testing.T) {
	// функции/каналы не сериализуются в JSON — спровоцируем ошибку.
	_, err := NewEvent(EventUserUpdated, func() {})
	assert.Error(t, err)
}

func TestEvent_UnmarshalPayload_OK(t *testing.T) {
	e, err := NewEvent(EventAdSold, AdPayload{AdID: 1, BuyerID: 2})
	require.NoError(t, err)

	var got AdPayload
	require.NoError(t, e.UnmarshalPayload(&got))
	assert.Equal(t, int64(1), got.AdID)
	assert.Equal(t, int64(2), got.BuyerID)
}

func TestEvent_UnmarshalPayload_BadJSON(t *testing.T) {
	e := Event{Payload: []byte("not json")}
	var out UserPayload
	assert.Error(t, e.UnmarshalPayload(&out))
}

func TestTopicConstants(t *testing.T) {
	assert.Equal(t, "clover.auth.user-events", TopicAuthUserEvents)
	assert.Equal(t, "clover.catalog.ad-events", TopicCatalogAdEvents)
	assert.Equal(t, "user.updated", EventUserUpdated)
	assert.Equal(t, "user.deleted", EventUserDeleted)
	assert.Equal(t, "ad.sold", EventAdSold)
	assert.Equal(t, "ad.deleted", EventAdDeleted)
}
