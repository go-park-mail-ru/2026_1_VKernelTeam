package views_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/view"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/views"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/views/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func defaultCfg() views.Config {
	return views.Config{
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		BlockTimeout:  time.Second,
	}
}

type env struct {
	cache    *mocks.MockViewCache
	producer *mocks.MockViewProducer
	consumer *mocks.MockViewConsumer
	storage  *mocks.MockViewStorage
	uc       *views.Views
}

func setup(t *testing.T) *env {
	t.Helper()
	ctrl := gomock.NewController(t)
	e := &env{
		cache:    mocks.NewMockViewCache(ctrl),
		producer: mocks.NewMockViewProducer(ctrl),
		consumer: mocks.NewMockViewConsumer(ctrl),
		storage:  mocks.NewMockViewStorage(ctrl),
	}
	e.uc = views.New(discardLogger(), e.cache, e.producer, e.consumer, e.storage, defaultCfg())
	return e
}

func int64Ptr(v int64) *int64 { return &v }

// ─── RecordView ──────────────────────────────────────────────────────────────

func TestRecordView_NewView_CacheHit(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	// dedup: новый просмотр
	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(10), "user:42").Return(true, nil)

	// прогрев кэша перед INCR — cache hit
	e.cache.EXPECT().GetCount(ctx, int64(10)).Return(int64(99), true, nil)

	// INCR
	e.cache.EXPECT().IncrementCount(ctx, int64(10)).Return(int64(100), nil)

	// publish best-effort
	e.producer.EXPECT().Publish(ctx, gomock.Any()).Return(nil)

	// финальный getViewsCount — cache hit
	e.cache.EXPECT().GetCount(ctx, int64(10)).Return(int64(100), true, nil)

	count, err := e.uc.RecordView(ctx, 10, int64Ptr(42), "dev-123")

	require.NoError(t, err)
	assert.Equal(t, int64(100), count)
}

func TestRecordView_DuplicateView(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(10), "device:dev-abc").Return(false, nil)

	// не новый — сразу возвращаем счётчик
	e.cache.EXPECT().GetCount(ctx, int64(10)).Return(int64(50), true, nil)

	count, err := e.uc.RecordView(ctx, 10, nil, "dev-abc")

	require.NoError(t, err)
	assert.Equal(t, int64(50), count)
}

func TestRecordView_DedupError_FallbackToCount(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(5), "device:d1").Return(false, errors.New("redis down"))

	// при ошибке dedup — возвращаем счётчик
	e.cache.EXPECT().GetCount(ctx, int64(5)).Return(int64(30), true, nil)

	count, err := e.uc.RecordView(ctx, 5, nil, "d1")

	require.NoError(t, err)
	assert.Equal(t, int64(30), count)
}

func TestRecordView_NewView_CacheMiss_LoadFromDB(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(7), "device:d1").Return(true, nil)

	// прогрев: cache miss → DB → SetCount
	e.cache.EXPECT().GetCount(ctx, int64(7)).Return(int64(0), false, nil)
	e.storage.EXPECT().GetViewsCount(ctx, int64(7)).Return(int64(200), nil)
	e.cache.EXPECT().SetCount(ctx, int64(7), int64(200)).Return(nil)

	e.cache.EXPECT().IncrementCount(ctx, int64(7)).Return(int64(201), nil)
	e.producer.EXPECT().Publish(ctx, gomock.Any()).Return(nil)

	// финальный getViewsCount — cache hit после прогрева
	e.cache.EXPECT().GetCount(ctx, int64(7)).Return(int64(201), true, nil)

	count, err := e.uc.RecordView(ctx, 7, nil, "d1")

	require.NoError(t, err)
	assert.Equal(t, int64(201), count)
}

func TestRecordView_IncrementError_StillReturnsCount(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(1), "device:d1").Return(true, nil)

	// прогрев — cache hit
	e.cache.EXPECT().GetCount(ctx, int64(1)).Return(int64(10), true, nil)

	// INCR fails
	e.cache.EXPECT().IncrementCount(ctx, int64(1)).Return(int64(0), errors.New("OOM"))

	// publish всё равно вызывается
	e.producer.EXPECT().Publish(ctx, gomock.Any()).Return(nil)

	// финальный count
	e.cache.EXPECT().GetCount(ctx, int64(1)).Return(int64(10), true, nil)

	count, err := e.uc.RecordView(ctx, 1, nil, "d1")

	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

func TestRecordView_PublishError_StillReturnsCount(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(1), "device:d1").Return(true, nil)
	e.cache.EXPECT().GetCount(ctx, int64(1)).Return(int64(5), true, nil)
	e.cache.EXPECT().IncrementCount(ctx, int64(1)).Return(int64(6), nil)
	e.producer.EXPECT().Publish(ctx, gomock.Any()).Return(errors.New("stream full"))

	// всё равно возвращаем счётчик
	e.cache.EXPECT().GetCount(ctx, int64(1)).Return(int64(6), true, nil)

	count, err := e.uc.RecordView(ctx, 1, nil, "d1")

	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
}

func TestRecordView_DBError_OnCacheMiss(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	e.cache.EXPECT().CheckAndSetDedup(ctx, int64(3), "device:d1").Return(false, errors.New("fail"))

	// fallback getViewsCount: cache miss → DB error
	e.cache.EXPECT().GetCount(ctx, int64(3)).Return(int64(0), false, nil)
	e.storage.EXPECT().GetViewsCount(ctx, int64(3)).Return(int64(0), errors.New("connection lost"))

	count, err := e.uc.RecordView(ctx, 3, nil, "d1")

	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
}

// ─── RunConsumer / flushBatch ────────────────────────────────────────────────

func TestRunConsumer_FlushOnBatchSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := mocks.NewMockViewCache(ctrl)
	producer := mocks.NewMockViewProducer(ctrl)
	consumer := mocks.NewMockViewConsumer(ctrl)
	storage := mocks.NewMockViewStorage(ctrl)

	cfg := views.Config{
		BatchSize:     2,
		FlushInterval: time.Hour, // не сработает по таймеру
		BlockTimeout:  10 * time.Millisecond,
	}
	uc := views.New(discardLogger(), cache, producer, consumer, storage, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	now := time.Now()
	batch := []models.ViewEvent{
		{MessageID: "1-0", ProductID: 1, ViewedAt: now},
		{MessageID: "2-0", ProductID: 2, ViewedAt: now},
	}

	callCount := 0
	consumer.EXPECT().ReadBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, _ time.Duration) ([]models.ViewEvent, error) {
			callCount++
			if callCount == 1 {
				return batch, nil
			}
			cancel()
			return nil, ctx.Err()
		}).Times(2)

	storage.EXPECT().BatchInsertViews(gomock.Any(), []view.InsertEvent{
		{ProductID: 1, ViewedAt: now},
		{ProductID: 2, ViewedAt: now},
	}).Return(nil)

	consumer.EXPECT().Ack(gomock.Any(), []string{"1-0", "2-0"}).Return(nil)

	uc.RunConsumer(ctx)
}

func TestRunConsumer_FlushOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := mocks.NewMockViewCache(ctrl)
	producer := mocks.NewMockViewProducer(ctrl)
	consumer := mocks.NewMockViewConsumer(ctrl)
	storage := mocks.NewMockViewStorage(ctrl)

	cfg := views.Config{
		BatchSize:     100, // большой — не достигнем
		FlushInterval: time.Hour,
		BlockTimeout:  10 * time.Millisecond,
	}
	uc := views.New(discardLogger(), cache, producer, consumer, storage, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()

	callCount := 0
	consumer.EXPECT().ReadBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, _ time.Duration) ([]models.ViewEvent, error) {
			callCount++
			if callCount == 1 {
				return []models.ViewEvent{{MessageID: "1-0", ProductID: 1, ViewedAt: now}}, nil
			}
			cancel()
			return nil, ctx.Err()
		}).Times(2)

	// При отмене ctx — flush оставшегося батча
	storage.EXPECT().BatchInsertViews(gomock.Any(), []view.InsertEvent{
		{ProductID: 1, ViewedAt: now},
	}).Return(nil)

	consumer.EXPECT().Ack(gomock.Any(), []string{"1-0"}).Return(nil)

	uc.RunConsumer(ctx)
}

func TestRunConsumer_FlushDBError_NoAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := mocks.NewMockViewCache(ctrl)
	producer := mocks.NewMockViewProducer(ctrl)
	consumer := mocks.NewMockViewConsumer(ctrl)
	storage := mocks.NewMockViewStorage(ctrl)

	cfg := views.Config{
		BatchSize:     1,
		FlushInterval: time.Hour,
		BlockTimeout:  10 * time.Millisecond,
	}
	uc := views.New(discardLogger(), cache, producer, consumer, storage, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()

	callCount := 0
	consumer.EXPECT().ReadBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, _ time.Duration) ([]models.ViewEvent, error) {
			callCount++
			if callCount == 1 {
				return []models.ViewEvent{{MessageID: "1-0", ProductID: 1, ViewedAt: now}}, nil
			}
			cancel()
			return nil, ctx.Err()
		}).Times(2)

	// DB error — Ack НЕ вызывается
	storage.EXPECT().BatchInsertViews(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	// Ack не должен вызываться — не устанавливаем EXPECT

	uc.RunConsumer(ctx)
}
