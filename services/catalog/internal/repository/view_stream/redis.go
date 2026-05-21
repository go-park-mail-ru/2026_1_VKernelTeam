package viewstream

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
	"github.com/gomodule/redigo/redis"
)

// ViewStream управляет Redis Streams для событий просмотров.
type ViewStream struct {
	pool          *redis.Pool
	log           *slog.Logger
	streamKey     string
	consumerGroup string
	consumerName  string
}

// New создаёт ViewStream для публикации и чтения событий просмотров через Redis Streams.
func New(pool *redis.Pool, log *slog.Logger, streamKey, consumerGroup, consumerName string) *ViewStream {
	return &ViewStream{
		pool:          pool,
		log:           log,
		streamKey:     streamKey,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
	}
}

// EnsureConsumerGroup создаёт consumer group, если она не существует.
func (s *ViewStream) EnsureConsumerGroup() error {
	conn := s.pool.Get()
	defer func() { _ = conn.Close() }()

	// XGROUP CREATE key groupname $ MKSTREAM — создаёт группу и стрим
	_, err := conn.Do("XGROUP", "CREATE", s.streamKey, s.consumerGroup, "$", "MKSTREAM")
	if err != nil {
		// Игнорируем ошибку "BUSYGROUP Consumer Group name already exists"
		if redisErr, ok := err.(redis.Error); ok {
			if strings.Contains(string(redisErr), "BUSYGROUP") {
				return nil
			}
		}
		return fmt.Errorf("viewstream.EnsureConsumerGroup: %w", err)
	}
	return nil
}

// Publish отправляет событие просмотра в Redis Stream.
func (s *ViewStream) Publish(_ context.Context, event models.ViewEvent) error {
	conn := s.pool.Get()
	defer func() { _ = conn.Close() }()

	userIDStr := ""
	if event.UserID != nil {
		userIDStr = strconv.FormatInt(*event.UserID, 10)
	}

	_, err := conn.Do("XADD", s.streamKey, "*",
		"product_id", strconv.FormatInt(event.ProductID, 10),
		"user_id", userIDStr,
		"viewed_at", event.ViewedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("viewstream.Publish: %w", err)
	}
	return nil
}

// ReadBatch читает пачку событий из Redis Stream через consumer group.
func (s *ViewStream) ReadBatch(_ context.Context, count int64, blockTimeout time.Duration) ([]models.ViewEvent, error) {
	conn := s.pool.Get()
	defer func() { _ = conn.Close() }()

	blockMs := int(blockTimeout.Milliseconds())

	// XREADGROUP GROUP group consumer COUNT count BLOCK blockMs STREAMS key >
	reply, err := redis.Values(conn.Do("XREADGROUP",
		"GROUP", s.consumerGroup, s.consumerName,
		"COUNT", count,
		"BLOCK", blockMs,
		"STREAMS", s.streamKey, ">",
	))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil // таймаут, нет новых сообщений
		}
		return nil, fmt.Errorf("viewstream.ReadBatch: %w", err)
	}

	return s.parseStreamReply(reply)
}

// Ack подтверждает обработку сообщений.
func (s *ViewStream) Ack(_ context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	conn := s.pool.Get()
	defer func() { _ = conn.Close() }()

	args := make([]any, 0, len(messageIDs)+2)
	args = append(args, s.streamKey, s.consumerGroup)
	for _, id := range messageIDs {
		args = append(args, id)
	}

	_, err := conn.Do("XACK", args...)
	if err != nil {
		return fmt.Errorf("viewstream.Ack: %w", err)
	}
	return nil
}

// parseStreamReply парсит ответ XREADGROUP в список ViewEvent.
func (s *ViewStream) parseStreamReply(reply []any) ([]models.ViewEvent, error) {
	// reply = [ [streamKey, [[msgID, [field, value, ...]], ...]] ]
	if len(reply) == 0 {
		return nil, nil
	}

	// Первый (и единственный) стрим
	streamData, err := redis.Values(reply[0], nil)
	if err != nil {
		return nil, fmt.Errorf("viewstream.parseStreamReply: stream: %w", err)
	}
	if len(streamData) < 2 {
		return nil, nil
	}

	messages, err := redis.Values(streamData[1], nil)
	if err != nil {
		return nil, fmt.Errorf("viewstream.parseStreamReply: messages: %w", err)
	}

	events := make([]models.ViewEvent, 0, len(messages))
	for _, msg := range messages {
		msgParts, err := redis.Values(msg, nil)
		if err != nil {
			s.log.Warn("skip malformed message", slog.String("error", err.Error()))
			continue
		}
		if len(msgParts) < 2 {
			continue
		}

		msgID, err := redis.String(msgParts[0], nil)
		if err != nil {
			continue
		}

		fields, err := redis.StringMap(msgParts[1], nil)
		if err != nil {
			s.log.Warn("skip message with bad fields", slog.String("id", msgID), slog.String("error", err.Error()))
			continue
		}

		event := models.ViewEvent{MessageID: msgID}

		if v, ok := fields["product_id"]; ok {
			event.ProductID, _ = strconv.ParseInt(v, 10, 64)
		}
		if v, ok := fields["user_id"]; ok && v != "" {
			uid, err := strconv.ParseInt(v, 10, 64)
			if err == nil {
				event.UserID = &uid
			}
		}
		if v, ok := fields["viewed_at"]; ok {
			event.ViewedAt, _ = time.Parse(time.RFC3339, v)
		}

		events = append(events, event)
	}

	return events, nil
}
