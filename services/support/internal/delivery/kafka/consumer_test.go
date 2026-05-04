package kafka

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewConsumer_RegistersHandler(t *testing.T) {
	c := NewConsumer([]string{"localhost:9092"}, "test-group", nil, newLogger())
	require.NotNil(t, c)
	// Close может вернуть ошибку, если broker недоступен — нам важен только сам факт вызова.
	_ = c.Close()
}

func TestHandleUserDeleted_OK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := &Consumer{pool: mock, log: newLogger()}

	mock.ExpectExec(`UPDATE support_ticket SET user_id = 0 WHERE user_id = \$1`).
		WithArgs(int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE support_message SET user_id = 0 WHERE user_id = \$1`).
		WithArgs(int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	event, err := sharedkafka.NewEvent(sharedkafka.EventUserDeleted, sharedkafka.UserPayload{UserID: 42})
	require.NoError(t, err)

	err = c.handleUserDeleted(context.Background(), event)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleUserDeleted_BadPayload(t *testing.T) {
	c := &Consumer{pool: nil, log: newLogger()}

	bad := sharedkafka.Event{
		EventType: sharedkafka.EventUserDeleted,
		Payload:   []byte("not json"),
	}

	err := c.handleUserDeleted(context.Background(), bad)
	assert.Error(t, err)
}

func TestHandleUserDeleted_TicketUpdateFails(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := &Consumer{pool: mock, log: newLogger()}

	mock.ExpectExec(`UPDATE support_ticket`).
		WithArgs(int64(7)).
		WillReturnError(errors.New("db error"))

	event, err := sharedkafka.NewEvent(sharedkafka.EventUserDeleted, sharedkafka.UserPayload{UserID: 7})
	require.NoError(t, err)

	err = c.handleUserDeleted(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tickets")
}

func TestHandleUserDeleted_MessageUpdateFails(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := &Consumer{pool: mock, log: newLogger()}

	mock.ExpectExec(`UPDATE support_ticket`).
		WithArgs(int64(7)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE support_message`).
		WithArgs(int64(7)).
		WillReturnError(errors.New("db error"))

	event, err := sharedkafka.NewEvent(sharedkafka.EventUserDeleted, sharedkafka.UserPayload{UserID: 7})
	require.NoError(t, err)

	err = c.handleUserDeleted(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages")
}
