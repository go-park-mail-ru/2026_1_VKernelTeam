# Auth & Users Service (Егор)

## Что переносить сюда

### Из `internal/`
| Откуда (монолит) | Куда (сервис) |
|-------------------|---------------|
| `internal/domain/models/user.go` | `internal/domain/models/` |
| `internal/domain/dto/auth.go` | `internal/domain/dto/` |
| `internal/delivery/handlers/auth.go` | `internal/delivery/handlers/` |
| `internal/usecase/auth/` | `internal/usecase/auth/` |
| `internal/repository/user/` | `internal/repository/user/` |
| `internal/repository/refresh/` | `internal/repository/refresh/` |
| `internal/repository/blacklist/` | `internal/repository/blacklist/` |

### Из `pkg/`
Используем через `pkg/shared/` (общие пакеты):
- jwt, validator, responser, sanitizer, middleware

### Миграции (`migrations/`)
Перенести из `internal/repository/postgres/migrations/`:
- Таблица `user`
- Таблица `refresh_token`

### gRPC сервер
Реализовать в `internal/delivery/grpc/`:
- `ValidateToken` — валидация JWT
- `GetUser` — данные пользователя по ID
- `GetUsersByIDs` — пакетное получение
- `CheckRole` — проверка роли

### Kafka producer
Топик: `clover.auth.user-events`
- `user.deleted`
- `user.updated`

### Инфраструктура
- Redis: blacklist токенов, кэш сессий
- S3: аватары пользователей
