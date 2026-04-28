# Support Service (Егор)

## Что переносить сюда

### Из `internal/`
| Откуда (монолит) | Куда (сервис) |
|-------------------|---------------|
| `internal/domain/models/support_ticket.go` | `internal/domain/models/` |
| `internal/domain/models/support_message.go` | `internal/domain/models/` |
| `internal/domain/dto/support_ticket.go` | `internal/domain/dto/` |
| `internal/domain/dto/support_message.go` | `internal/domain/dto/` |
| `internal/delivery/handlers/support_ticket.go` | `internal/delivery/handlers/` |
| `internal/usecase/support_ticket/` | `internal/usecase/support_ticket/` |
| `internal/usecase/support_message/` | `internal/usecase/support_message/` |
| `internal/repository/support_ticket/` | `internal/repository/support_ticket/` |
| `internal/repository/support_message/` | `internal/repository/support_message/` |

### Из `pkg/`
Используем через `pkg/shared/`:
- jwt, validator, responser, sanitizer, middleware

### Миграции (`migrations/`)
Перенести из `internal/repository/postgres/migrations/`:
- Таблица `support_ticket`
- Таблица `support_message`

### gRPC клиенты
- `AuthService.ValidateToken` — проверка токенов в middleware
- `AuthService.CheckRole` — проверка роли support/admin

### Kafka consumer
Топик: `clover.auth.user-events`
- `user.deleted` — анонимизация тикетов удалённого пользователя
