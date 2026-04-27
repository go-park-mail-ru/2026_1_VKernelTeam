package views

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/view"
)

// ViewCache описывает интерфейс кэша просмотров.
type ViewCache interface {
	CheckAndSetDedup(ctx context.Context, productID int64, identifier string) (bool, error)
	IncrementCount(ctx context.Context, productID int64) (int64, error)
	GetCount(ctx context.Context, productID int64) (int64, bool, error)
	SetCount(ctx context.Context, productID int64, count int64) error
}

// ViewProducer описывает интерфейс отправки событий в стрим.
type ViewProducer interface {
	Publish(ctx context.Context, event models.ViewEvent) error
}

// ViewConsumer описывает интерфейс чтения событий из стрима.
type ViewConsumer interface {
	ReadBatch(ctx context.Context, count int64, blockTimeout time.Duration) ([]models.ViewEvent, error)
	Ack(ctx context.Context, messageIDs []string) error
}

// ViewStorage описывает интерфейс хранилища просмотров.
type ViewStorage interface {
	BatchInsertViews(ctx context.Context, events []view.InsertEvent) error
	GetViewsCount(ctx context.Context, productID int64) (int64, error)
}

// Config содержит настройки для usecase просмотров.
type Config struct {
	BatchSize     int64
	FlushInterval time.Duration
	BlockTimeout  time.Duration
}

// Views реализует бизнес-логику счётчика просмотров.
type Views struct {
	log      *slog.Logger
	cache    ViewCache
	producer ViewProducer
	consumer ViewConsumer
	storage  ViewStorage
	cfg      Config
	sf       singleflight.Group
}

func New(
	log *slog.Logger,
	cache ViewCache,
	producer ViewProducer,
	consumer ViewConsumer,
	storage ViewStorage,
	cfg Config,
) *Views {
	return &Views{
		log:      log,
		cache:    cache,
		producer: producer,
		consumer: consumer,
		storage:  storage,
		cfg:      cfg,
	}
}

// RecordView фиксирует просмотр объявления и возвращает актуальный счётчик.
func (v *Views) RecordView(ctx context.Context, productID int64, userID *int64, deviceID string) (int64, error) {
	// Формируем идентификатор для дедупликации
	identifier := fmt.Sprintf("device:%s", deviceID)
	if userID != nil {
		identifier = fmt.Sprintf("user:%d", *userID)
	}

	// Дедупликация
	isNew, err := v.cache.CheckAndSetDedup(ctx, productID, identifier)
	if err != nil {
		v.log.ErrorContext(ctx, "dedup check failed",
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		// При ошибке Redis — возвращаем счётчик без записи
		return v.getViewsCount(ctx, productID)
	}

	if isNew {
		// Прогреваем кэш перед INCR, чтобы INCR не создал ключ со значением 1
		// при холодном кэше (после рестарта Redis)
		_, _ = v.getViewsCount(ctx, productID)

		// Инкрементируем кэш
		if _, err := v.cache.IncrementCount(ctx, productID); err != nil {
			v.log.ErrorContext(ctx, "increment count failed",
				slog.Int64("product_id", productID),
				slog.String("error", err.Error()),
			)
		}

		// Отправляем событие в стрим (best-effort)
		event := models.ViewEvent{
			ProductID: productID,
			UserID:    userID,
			ViewedAt:  time.Now(),
		}
		if err := v.producer.Publish(ctx, event); err != nil {
			v.log.ErrorContext(ctx, "publish view event failed",
				slog.Int64("product_id", productID),
				slog.String("error", err.Error()),
			)
		}
	}

	return v.getViewsCount(ctx, productID)
}

// getViewsCount возвращает счётчик из кэша, при cache miss — из БД через singleflight.
func (v *Views) getViewsCount(ctx context.Context, productID int64) (int64, error) {
	count, found, err := v.cache.GetCount(ctx, productID)
	if err != nil {
		v.log.ErrorContext(ctx, "cache get count failed",
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
	}
	if found {
		return count, nil
	}

	// Cache miss — загружаем из БД через singleflight
	key := fmt.Sprintf("views_count:%d", productID)
	result, err, _ := v.sf.Do(key, func() (any, error) {
		dbCount, err := v.storage.GetViewsCount(ctx, productID)
		if err != nil {
			return int64(0), err
		}
		// Прогреваем кэш
		if setErr := v.cache.SetCount(ctx, productID, dbCount); setErr != nil {
			v.log.ErrorContext(ctx, "cache set count failed",
				slog.Int64("product_id", productID),
				slog.String("error", setErr.Error()),
			)
		}
		return dbCount, nil
	})
	if err != nil {
		return 0, fmt.Errorf("views.getViewsCount: %w", err)
	}
	return result.(int64), nil
}

// RunConsumer запускает фоновый воркер для обработки событий из Redis Stream.
// Блокирующий вызов — запускать в горутине. Останавливается при отмене ctx.
func (v *Views) RunConsumer(ctx context.Context) {
	v.log.Info("views consumer started")
	defer v.log.Info("views consumer stopped")

	var batch []models.ViewEvent
	lastFlush := time.Now()

	for {
		// Проверяем отмену контекста
		if ctx.Err() != nil {
			if len(batch) > 0 {
				v.flushBatch(context.Background(), batch)
			}
			return
		}

		// ReadBatch блокируется на blockTimeout (1s), затем возвращается —
		// это даёт естественный цикл проверки ctx и flush таймаута
		events, err := v.consumer.ReadBatch(ctx, v.cfg.BatchSize-int64(len(batch)), v.cfg.BlockTimeout)
		if err != nil {
			if ctx.Err() != nil {
				if len(batch) > 0 {
					v.flushBatch(context.Background(), batch)
				}
				return
			}
			v.log.Error("read batch failed", slog.String("error", err.Error()))
			time.Sleep(1 * time.Second)
			continue
		}

		batch = append(batch, events...)

		// Flush по размеру ИЛИ по таймауту
		if int64(len(batch)) >= v.cfg.BatchSize || (len(batch) > 0 && time.Since(lastFlush) >= v.cfg.FlushInterval) {
			v.flushBatch(ctx, batch)
			batch = batch[:0]
			lastFlush = time.Now()
		}
	}
}

// flushBatch записывает пачку событий в БД и подтверждает их в стриме.
func (v *Views) flushBatch(ctx context.Context, batch []models.ViewEvent) {
	if len(batch) == 0 {
		return
	}

	insertEvents := make([]view.InsertEvent, len(batch))
	messageIDs := make([]string, len(batch))

	for i, e := range batch {
		insertEvents[i] = view.InsertEvent{
			ProductID: e.ProductID,
			UserID:    e.UserID,
			ViewedAt:  e.ViewedAt,
		}
		messageIDs[i] = e.MessageID
	}

	if err := v.storage.BatchInsertViews(ctx, insertEvents); err != nil {
		v.log.Error("flush batch failed",
			slog.Int("batch_size", len(batch)),
			slog.String("error", err.Error()),
		)
		return // Не ACK-аем — события будут перечитаны
	}

	// COMMIT прошёл успешно — теперь XACK
	if err := v.consumer.Ack(ctx, messageIDs); err != nil {
		v.log.Error("ack failed",
			slog.Int("batch_size", len(batch)),
			slog.String("error", err.Error()),
		)
	}

	v.log.Debug("batch flushed",
		slog.Int("count", len(batch)),
	)
}
