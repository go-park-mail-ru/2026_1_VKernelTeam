# Commerce Service (Николай)

## Что переносить сюда

### Из `internal/`
| Откуда (монолит) | Куда (сервис) |
|-------------------|---------------|
| `internal/domain/models/cart.go` | `internal/domain/models/` |
| `internal/domain/models/chat.go` | `internal/domain/models/` |
| `internal/domain/dto/cart.go` | `internal/domain/dto/` |
| `internal/domain/dto/chat.go` | `internal/domain/dto/` |
| `internal/delivery/handlers/cart.go` | `internal/delivery/handlers/` |
| `internal/delivery/handlers/chat.go` | `internal/delivery/handlers/` |
| `internal/usecase/cart/` | `internal/usecase/cart/` |
| `internal/usecase/chat/` | `internal/usecase/chat/` |
| `internal/repository/cart/` | `internal/repository/cart/` |
| `internal/repository/chat/` | `internal/repository/chat/` |

### Из `pkg/`
Используем через `pkg/shared/`:
- jwt, validator, responser, sanitizer, middleware

### Миграции (`migrations/`)
Перенести из `internal/repository/postgres/migrations/`:
- Таблица `cart_item`
- Таблица `order`
- Таблица `order_item`
- Таблица `chat`
- Таблица `message`

### gRPC клиенты
- `AuthService.ValidateToken` — проверка токенов в middleware
- `AuthService.GetUser` / `GetUsersByIDs` — данные покупателя/продавца в чатах
- `CatalogService.GetAd` — информация о товаре
- `CatalogService.CheckAdStatus` — валидация при добавлении в корзину
- `CatalogService.UpdateAdStatus` — смена статуса при покупке

### Kafka consumer
Топик: `clover.catalog.ad-events`
- `ad.deleted` — удалить товар из всех корзин
