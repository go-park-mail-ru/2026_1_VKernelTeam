# Catalog Service (Николай)

## Что переносить сюда

### Из `internal/`
| Откуда (монолит) | Куда (сервис) |
|-------------------|---------------|
| `internal/domain/models/ad.go` | `internal/domain/models/` |
| `internal/domain/models/view.go` | `internal/domain/models/` |
| `internal/domain/models/characteristic.go` | `internal/domain/models/` |
| `internal/domain/dto/ad.go` | `internal/domain/dto/` |
| `internal/delivery/handlers/ads.go` | `internal/delivery/handlers/` |
| `internal/delivery/handlers/views.go` | `internal/delivery/handlers/` |
| `internal/usecase/ads/` | `internal/usecase/ads/` |
| `internal/usecase/views/` | `internal/usecase/views/` |
| `internal/repository/ad/` | `internal/repository/ad/` |
| `internal/repository/view/` | `internal/repository/view/` |
| `internal/repository/view_cache/` | `internal/repository/view_cache/` |
| `internal/repository/view_stream/` | `internal/repository/view_stream/` |

### Из `pkg/`
Перенести в сервис (используется только здесь):
- `pkg/synonyms/` — синонимы для поиска
- `pkg/translit/` — транслитерация

Используем через `pkg/shared/`:
- jwt, validator, responser, sanitizer, middleware

### Миграции (`migrations/`)
Перенести из `internal/repository/postgres/migrations/`:
- Таблица `product`
- Таблица `product_image`
- Таблица `category`
- Таблица `category_characteristic`
- Таблица `product_characteristic`
- Таблица `product_custom_characteristic`
- Таблица `favorite`
- Таблица `product_view`
- Таблица `product_price_history`
- Таблица `product_status_history`

### gRPC сервер
Реализовать в `internal/delivery/grpc/`:
- `GetAd` — объявление по ID
- `GetAdsByIDs` — пакетное получение
- `CheckAdStatus` — проверка статуса
- `UpdateAdStatus` — смена статуса (при покупке)

### gRPC клиенты
- `AuthService.ValidateToken` — проверка токенов в middleware

### Kafka producer
Топик: `clover.catalog.ad-events`
- `ad.sold`
- `ad.deleted`

### Kafka consumer
Топик: `clover.auth.user-events`
- `user.updated` — инвалидация кэша данных продавцов

### Инфраструктура
- Redis: дедупликация просмотров, Redis Streams (view pipeline), кэш
- S3: фото товаров
