package viewstream

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testStream   = "views:stream"
	testGroup    = "views-group"
	testConsumer = "worker-0"
)

func setup(t *testing.T) (*ViewStream, *redigomock.Conn) {
	t.Helper()
	mock := redigomock.NewConn()
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) { return mock, nil },
	}
	return New(pool, slog.Default(), testStream, testGroup, testConsumer), mock
}

// ─── EnsureConsumerGroup ─────────────────────────────────────────────────────

func TestEnsureConsumerGroup_Created(t *testing.T) {
	stream, mock := setup(t)

	mock.Command("XGROUP", "CREATE", testStream, testGroup, "$", "MKSTREAM").Expect("OK")

	err := stream.EnsureConsumerGroup()

	assert.NoError(t, err)
}

func TestEnsureConsumerGroup_AlreadyExists(t *testing.T) {
	stream, mock := setup(t)

	mock.Command("XGROUP", "CREATE", testStream, testGroup, "$", "MKSTREAM").
		ExpectError(redis.Error("BUSYGROUP Consumer Group name already exists"))

	err := stream.EnsureConsumerGroup()

	assert.NoError(t, err)
}

func TestEnsureConsumerGroup_Error(t *testing.T) {
	stream, mock := setup(t)

	mock.Command("XGROUP", "CREATE", testStream, testGroup, "$", "MKSTREAM").
		ExpectError(fmt.Errorf("connection refused"))

	err := stream.EnsureConsumerGroup()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EnsureConsumerGroup")
}

// ─── Publish ─────────────────────────────────────────────────────────────────

func TestPublish_WithUserID(t *testing.T) {
	stream, mock := setup(t)
	uid := int64(42)
	now := time.Now().Truncate(time.Second)

	mock.Command("XADD", testStream, "*",
		"product_id", "10",
		"user_id", "42",
		"viewed_at", now.Format(time.RFC3339),
	).Expect("1234567890-0")

	err := stream.Publish(context.Background(), models.ViewEvent{
		ProductID: 10,
		UserID:    &uid,
		ViewedAt:  now,
	})

	assert.NoError(t, err)
}

func TestPublish_WithoutUserID(t *testing.T) {
	stream, mock := setup(t)
	now := time.Now().Truncate(time.Second)

	mock.Command("XADD", testStream, "*",
		"product_id", "5",
		"user_id", "",
		"viewed_at", now.Format(time.RFC3339),
	).Expect("1234567890-1")

	err := stream.Publish(context.Background(), models.ViewEvent{
		ProductID: 5,
		UserID:    nil,
		ViewedAt:  now,
	})

	assert.NoError(t, err)
}

func TestPublish_RedisError(t *testing.T) {
	stream, mock := setup(t)

	mock.GenericCommand("XADD").ExpectError(fmt.Errorf("OOM"))

	err := stream.Publish(context.Background(), models.ViewEvent{
		ProductID: 1,
		ViewedAt:  time.Now(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Publish")
}

// ─── ReadBatch ───────────────────────────────────────────────────────────────

func TestReadBatch_Timeout(t *testing.T) {
	stream, mock := setup(t)

	mock.GenericCommand("XREADGROUP").ExpectError(redis.ErrNil)

	events, err := stream.ReadBatch(context.Background(), 10, time.Second)

	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestReadBatch_RedisError(t *testing.T) {
	stream, mock := setup(t)

	mock.GenericCommand("XREADGROUP").ExpectError(fmt.Errorf("NOGROUP"))

	events, err := stream.ReadBatch(context.Background(), 10, time.Second)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ReadBatch")
	assert.Nil(t, events)
}

// ─── Ack ─────────────────────────────────────────────────────────────────────

func TestAck_EmptyIDs(t *testing.T) {
	stream, _ := setup(t)

	err := stream.Ack(context.Background(), nil)

	assert.NoError(t, err)
}

func TestAck_Success(t *testing.T) {
	stream, mock := setup(t)

	mock.Command("XACK", testStream, testGroup, "1-0", "2-0").Expect(int64(2))

	err := stream.Ack(context.Background(), []string{"1-0", "2-0"})

	assert.NoError(t, err)
}

func TestAck_RedisError(t *testing.T) {
	stream, mock := setup(t)

	mock.GenericCommand("XACK").ExpectError(fmt.Errorf("ERR"))

	err := stream.Ack(context.Background(), []string{"1-0"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Ack")
}

// ─── parseStreamReply ────────────────────────────────────────────────────────

func TestParseStreamReply_EmptyReply(t *testing.T) {
	stream, _ := setup(t)

	events, err := stream.parseStreamReply([]any{})

	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestParseStreamReply_ValidMessages(t *testing.T) {
	stream, _ := setup(t)
	now := time.Now().Truncate(time.Second)

	// Эмулируем ответ Redis XREADGROUP:
	// [ [streamKey, [ [msgID, [k,v,k,v,...]], ... ]] ]
	reply := []any{
		[]any{
			[]byte(testStream),
			[]any{
				[]any{
					[]byte("100-0"),
					[]any{
						[]byte("product_id"), []byte("7"),
						[]byte("user_id"), []byte("42"),
						[]byte("viewed_at"), []byte(now.Format(time.RFC3339)),
					},
				},
				[]any{
					[]byte("101-0"),
					[]any{
						[]byte("product_id"), []byte("8"),
						[]byte("user_id"), []byte(""),
						[]byte("viewed_at"), []byte(now.Format(time.RFC3339)),
					},
				},
			},
		},
	}

	events, err := stream.parseStreamReply(reply)

	require.NoError(t, err)
	require.Len(t, events, 2)

	assert.Equal(t, "100-0", events[0].MessageID)
	assert.Equal(t, int64(7), events[0].ProductID)
	require.NotNil(t, events[0].UserID)
	assert.Equal(t, int64(42), *events[0].UserID)
	assert.Equal(t, now, events[0].ViewedAt)

	assert.Equal(t, "101-0", events[1].MessageID)
	assert.Equal(t, int64(8), events[1].ProductID)
	assert.Nil(t, events[1].UserID)
}
