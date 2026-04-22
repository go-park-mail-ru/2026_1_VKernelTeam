# Ad Views Counter — Design Spec

**Дата:** 2026-04-22
**Статус:** Утверждён
**Автор:** @virsi + Claude

## Обзор

Счётчик просмотров объявлений для маркетплейса Клевер. Фиксирует уникальные просмотры с дедупликацией через Redis, асинхронно записывает в PostgreSQL через Redis Streams, кэширует счётчик в Redis для мгновенного ответа.

## 1. API-контракт

**Эндпоинт:** `POST /ads/{id}/view`

**Заголовки:**
- `X-Device-ID: <UUID v4>` — обязательный
- Cookie сессии — опционально (если авторизован, берём user_id)

**Ответ:** `200 OK`
```json
{
  "views_count": 42
}
```

**Ошибки:**
- `400` — невалидный `id` или `X-Device-ID` не UUID v4
- `404` — объявление не найдено или удалено

`views_count` берётся из кэшированного значения в Redis (может отставать на интервал flush при cache miss fallback). Тело запроса отсутствует.

## 2. Дедупликация (Redis)

**Ключ:** `view:{product_id}:{identifier}`, где `identifier`:
- `user:{user_id}` — для авторизованных пользователей
- `device:{device_id}` — для анонимов

Примеры:
- `view:15:user:42`
- `view:15:device:550e8400-e29b-41d4-a716-446655440000`

**Логика:**
1. Формируем ключ
2. `SET key 1 EX <ttl> NX` — атомарная операция: устанавливает ключ только если его нет
3. Если SET вернул OK — просмотр новый, отправляем в очередь
4. Если SET вернул nil — дубликат, игнорируем

**TTL:** из конфига, по умолчанию 24 часа.

**Приоритет:** если пользователь авторизован — используем `user_id`, даже если `X-Device-ID` тоже пришёл. Это исключает двойной счёт при логине/логауте.

**Валидация X-Device-ID:** строгая на уровне Handler (Delivery layer):
- Принимаем только валидный UUID v4 (regex: `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, case-insensitive)
- Иначе — `400 Bad Request`
- Защита от переполнения Redis: длинные/мусорные строки отсекаются до попадания в Redis-ключ

**Ограничение payload:** тело POST-запроса должно быть пустым. На уровне Nginx/API Gateway рекомендуется `client_max_body_size 1k` для этого эндпоинта.

## 3. Очередь (Redis Streams) + Background Worker

### Почему Redis Streams, а не Kafka

- Redis уже есть в инфраструктуре (для дедупликации)
- Для текущих нагрузок маркетплейса Redis Streams более чем достаточен
- При переходе на микросервисы замена Redis Streams -> Kafka будет простой: меняется только адаптер, интерфейс producer/consumer остаётся тем же

### Событие

```
Stream: views:events
Entry: {
    "product_id": "15",
    "user_id":    "42",     // пустая строка для анонимов
    "viewed_at":  "2026-04-22T12:00:00Z"
}
```

### Producer (в usecase)

1. Дедупликация через Redis SET NX (секция 2)
2. Если просмотр новый — `XADD views:events * product_id 15 user_id 42 viewed_at ...`
3. Сразу ответ `200 OK`
4. Если XADD упал — лог error, ответ клиенту всё равно 200 (best-effort)

### Consumer (Background Worker)

- Отдельная горутина, запускается при старте приложения
- Использует **Consumer Groups** (XREADGROUP, не XREAD) — гарантирует at-least-once доставку, возможность нескольких воркеров, автоматический offset
- `XREADGROUP GROUP views-consumer worker-1 COUNT <batch_size> BLOCK <block_timeout> STREAMS views:events >`

**Триггеры flush (условие ИЛИ):**
- Набралось `batch_size` событий (по умолчанию 100), **ИЛИ**
- Прошло `flush_interval` времени (по умолчанию 2s) с момента последнего flush
- Это гарантирует, что при низком трафике (ночью) просмотры не зависают в стриме

**Порядок операций (строгий):**
1. XREADGROUP — читаем пачку из стрима
2. BEGIN транзакции в PostgreSQL
3. Batch INSERT в `product_view`
4. UPDATE `product SET views_count = views_count + N` (группировка по product_id)
5. COMMIT транзакции
6. **Только после успешного COMMIT** — `XACK` для подтверждения обработки
7. При ошибке на шагах 2-5 — ROLLBACK, события остаются в Pending Entries List (PEL), будут перечитаны

**Обработка зависших событий:** периодическая проверка XPENDING + XCLAIM для событий, которые были прочитаны, но не подтверждены (воркер упал между READ и ACK). Таймаут claim: 30s.

**Graceful shutdown:** worker завершает текущий batch, дожидается COMMIT, ACK-ает обработанные события. Необработанные остаются в стриме.

## 4. Кэширование views_count + Денормализация

### Денормализованный счётчик

Колонка `views_count bigint NOT NULL DEFAULT 0` в таблице `product`.

**Background Worker** при batch INSERT в `product_view` одновременно обновляет счётчик:

```sql
UPDATE product SET views_count = views_count + <count> WHERE id = <product_id>;
```

Группировка по `product_id` внутри пачки — один UPDATE на объявление.

**Существующие запросы** (GET /ads, GET /ads/{id}) заменяют `COUNT(DISTINCT pv.id)` через JOIN на чтение `p.views_count`. Убираем `LEFT JOIN product_view` — запросы становятся легче.

### Redis-кэш счётчика

- `INCR views:count:{product_id}` при новом просмотре — ответ клиенту мгновенный
- TTL 48h, обновляется при каждом INCR

### Cache miss

Применяется в обоих сценариях — и при новом просмотре (после INCR), и при дубликате (при GET count). При отсутствии ключа `views:count:{product_id}` в Redis:
- Через singleflight выполняем `SELECT views_count FROM product WHERE id = $1`
- Результат записывается в Redis с TTL
- Если это новый просмотр — поверх выполняется INCR

### Защита от thundering herd

`golang.org/x/sync/singleflight` в usecase — один горутин идёт в БД, остальные ждут его результат in-process. Достаточно для одного инстанса. Распределённая блокировка в Redis нужна только при нескольких подах.

## 5. Слои архитектуры и интерфейсы

### Repository layer — `internal/repository/view/`

```go
type ViewStorage interface {
    BatchInsertViews(ctx context.Context, events []ViewEvent) error
    GetViewsCount(ctx context.Context, productID int64) (int64, error)
}
```

### Cache layer — `internal/repository/view_cache/`

```go
type ViewCache interface {
    CheckAndSetDedup(ctx context.Context, productID int64, identifier string) (bool, error)
    IncrementCount(ctx context.Context, productID int64) (int64, error)
    GetCount(ctx context.Context, productID int64) (int64, bool, error)
    SetCount(ctx context.Context, productID int64, count int64) error
}
```

### Stream layer — `internal/repository/view_stream/`

```go
type ViewProducer interface {
    Publish(ctx context.Context, event ViewEvent) error
}

type ViewConsumer interface {
    ReadBatch(ctx context.Context, count int64) ([]ViewEvent, error)
    Ack(ctx context.Context, ids []string) error
}
```

### Usecase layer — `internal/usecase/views/`

```go
type Views struct {
    log       *slog.Logger
    cache     ViewCache
    producer  ViewProducer
    storage   ViewStorage
    sf        singleflight.Group
}

func (v *Views) RecordView(ctx context.Context, productID int64, userID *int64, deviceID string) (int64, error)
func (v *Views) RunConsumer(ctx context.Context)
```

### Delivery layer

```go
func (h *AdsHandlers) HandleRecordView(w http.ResponseWriter, r *http.Request)
```

Извлекает `product_id` из URL, `user_id` из сессии (если есть), `X-Device-ID` из заголовка.

### Изменения в существующем коде

- `repository/ad/postgres.go` — убираем `LEFT JOIN product_view`, читаем `p.views_count` напрямую
- `handlers/handlers.go` — добавляем интерфейс `Views` в `Services`
- Роутер — регистрируем `POST /ads/{id}/view`

## 6. Миграция данных

**Новая миграция** `000002_add_views_count.up.sql`:

```sql
ALTER TABLE product ADD COLUMN views_count bigint NOT NULL DEFAULT 0;

UPDATE product p
SET views_count = (
    SELECT COUNT(*)
    FROM product_view pv
    WHERE pv.product_id = p.id
);

CREATE INDEX idx_product_view_product_viewed
    ON product_view (product_id, viewed_at);
```

**Down-миграция** `000002_add_views_count.down.sql`:

```sql
ALTER TABLE product DROP COLUMN views_count;
```

## 7. Индексы базы данных (добавить в миграцию)

### Таблица `product_view`

Аналитическая таблица — оптимизируем под запись и аналитические выборки:

```sql
CREATE INDEX idx_product_view_product_viewed
    ON product_view (product_id, viewed_at);
```

- Покрывает запросы на построение графиков просмотров по времени для владельца объявления
- **Не** добавляем уникальный индекс — дедупликация на уровне Redis, таблица append-only
- Индекс по `user_id` не нужен — нет use case для выборки "все просмотры пользователя"

### Таблица `product`

- UPDATE `views_count` идёт по первичному ключу `WHERE id = $1` — index scan, блокировка только одной строки (row-level lock), без table lock
- Дополнительных индексов не требуется

## 8. Конфигурация

```yaml
views:
  dedup_ttl: 24h
  stream_key: "views:events"
  consumer_group: "views-consumer"
  batch_size: 100
  flush_interval: 2s        # Макс. задержка flush при низком трафике
  block_timeout: 1s
  count_cache_ttl: 48h
  claim_timeout: 30s         # Таймаут для XCLAIM зависших событий
```

## Диаграмма потока данных

```
Client
  │
  ▼
POST /ads/{id}/view (X-Device-ID header)
  │
  ▼
Handler — валидация UUID v4, извлечение user_id из сессии
  │
  ▼
Usecase.RecordView()
  │
  ├─► Redis: SET view:{pid}:{id} NX EX ttl
  │     │
  │     ├─ nil (дубликат) ──► Redis: GET views:count:{pid} ──► 200 { views_count }
  │     │
  │     └─ OK (новый) ──┬──► Redis: INCR views:count:{pid}
  │                      └──► Redis: XADD views:events ...
  │                                        │
  │                                        ▼ 200 { views_count }
  │
  ▼ (Background Worker — отдельная горутина)
  │
  XREADGROUP ... BLOCK 1000
  │
  ▼
  Batch INSERT into product_view
  + UPDATE product SET views_count = views_count + N
  │
  ▼
  XACK
```
