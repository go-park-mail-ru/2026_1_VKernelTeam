# Kafka Shared Client (Егор)

Общий Go-клиент для работы с Kafka. Используется всеми сервисами.

## Структура

```
kafka/
├── producer.go    # обёртка над segmentio/kafka-go Writer
├── consumer.go    # consumer group с graceful shutdown
├── config.go      # конфигурация (brokers, topics, group ID)
└── events.go      # общие типы событий
```

## Топики

| Топик | Продюсер | Консьюмеры | Partition key |
|-------|----------|------------|---------------|
| `clover.auth.user-events` | Auth | Support, Catalog | `user_id` |
| `clover.catalog.ad-events` | Catalog | Commerce | `ad_id` |

## Формат сообщений

```json
{
  "event_type": "user.deleted",
  "event_id": "uuid-v4",
  "timestamp": "2026-04-28T12:00:00Z",
  "payload": { ... }
}
```
