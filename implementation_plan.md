# JWT Refresh Token + Redis (redigo) — план внедрения

Сейчас в проекте есть только **access-токен** (JWT, 24 ч) и **InMemory-блэклист** для отозванных токенов при выходе.
Цель: добавить **refresh-токен** (долгоживущий, хранится в Redis) и перенести хранилище отозванных access-токенов тоже в Redis, заменив [InMemory](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/repository/blacklist/inmemory.go#10-18) на Redis-репозиторий через `redigo`.

---

## Схема работы после внедрения

```
Login  → выдаём access (15 min) + refresh (7 days, в куке httpOnly)
Refresh → ротируем refresh (старый удаляем из Redis, новый пишем) + выдаём новый access
Logout  → удаляем refresh из Redis + пишем jti access-токена в Redis blacklist
Auth middleware → проверяем: токен не в blacklist?
```

> [!IMPORTANT]
> Refresh токен является непрозрачной случайной строкой (UUID/crypto rand), **не JWT**.
> Хранится в Redis с ключом `refresh:<token>` → значение `userID`, TTL = 7 дней.
> Access-токен остаётся JWT HS256 (15 мин).

---

## User Review Required

> [!WARNING]
> Текущий `TokenTTL` (24 ч) становится TTL **access-токена** (рекомендую уменьшить до 15 мин).
> Отдельно нужно будет задать `REFRESH_TTL` (рекомендую 7 дней).

> [!IMPORTANT]
> Refresh-токен передаётся **только через httpOnly Cookie** (`refresh_token`).
> Access-токен — тоже через Cookie (`access_token`), как сейчас.

---

## Proposed Changes

### 1. Зависимости

#### [MODIFY] [go.mod](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/go.mod)

Добавить:
```
github.com/gomodule/redigo v1.9.2
```

Команда:
```bash
go get github.com/gomodule/redigo@v1.9.2
```

---

### 2. Config

#### [MODIFY] [config.go](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/config/config.go)

Добавить поля в [Config](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/config/config.go#17-25) и `rawConfig`:

```go
RedisAddr   string        // из .env: REDIS_ADDR=redis:6379
RefreshTTL  time.Duration // из JSON: "refresh_ttl": "168h"
```

В [MustLoadConfig](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/config/config.go#31-87) добавить чтение:
```go
redisAddr := os.Getenv("REDIS_ADDR")
if redisAddr == "" {
    redisAddr = "localhost:6379" // дефолт для локалки
}
```

#### [MODIFY] [local.json](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/config/local.json)

```json
{
  "env": "local",
  "token_ttl": "15m",
  "refresh_ttl": "168h",
  "http": { "port": 8000 },
  "cleanup_interval": "5m"
}
```

#### [MODIFY] [.env](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/.env)

Добавить:
```
REDIS_ADDR=localhost:6379
```

---

### 3. Redis-репозиторий

#### [NEW] `internal/repository/redis/redis.go`

Новый пакет. Реализует два интерфейса:

**[TokenRevoker](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/usecase/auth/auth.go#25-28)** (заменяет InMemory blacklist):
```go
// Add(jti string, exp time.Time)
// Проверяет, отозван ли access-токен по jti
// Ключ: "blacklist:<jti>", TTL = время до exp
```

**`RefreshStorage`** (новый):
```go
// SaveRefresh(ctx, token string, userID int64, ttl time.Duration) error
// GetRefresh(ctx, token string) (userID int64, err error)
// DeleteRefresh(ctx, token string) error
```

Структура:
```go
type Storage struct {
    pool *redigo.Pool
}

func New(addr string) *Storage {
    pool := &redigo.Pool{
        MaxIdle:     10,
        IdleTimeout: 240 * time.Second,
        Dial: func() (redigo.Conn, error) {
            return redigo.Dial("tcp", addr)
        },
    }
    return &Storage{pool: pool}
}

func (s *Storage) Close() { s.pool.Close() }
```

---

### 4. Auth Usecase

#### [MODIFY] [auth.go](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/usecase/auth/auth.go)

Добавить интерфейс `RefreshStorage`:
```go
type RefreshStorage interface {
    SaveRefresh(ctx context.Context, token string, userID int64, ttl time.Duration) error
    GetRefresh(ctx context.Context, token string) (int64, error)
    DeleteRefresh(ctx context.Context, token string) error
}
```

Добавить поле в [Auth](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/usecase/auth/auth.go#41-48):
```go
refreshStorage RefreshStorage
refreshTTL     time.Duration
```

Добавить/изменить методы:

- **[Login](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/usecase/auth/auth.go#74-105)** — генерирует и access, и refresh токен, сохраняет refresh в Redis
- **`Refresh(ctx, refreshToken string) (newAccess, newRefresh string, err error)`** — ротация:
  1. Получить userID из Redis по refresh
  2. Удалить старый refresh
  3. Создать новый refresh (UUID), сохранить в Redis
  4. Создать новый access JWT
- **[Logout](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/usecase/auth/auth.go#143-158)** — удаляет refresh из Redis + добавляет jti access в blacklist

---

### 5. Delivery (HTTP handlers)

#### [MODIFY] handlers (нужно найти файлы в `internal/delivery/handlers/`)

- **`POST /auth/login`** — ставить **две** куки: `access_token` (15 мин) + `refresh_token` (7 дней, httpOnly)
- **`POST /auth/refresh`** (новый endpoint) — читает куку `refresh_token`, вызывает `auth.Refresh`, обновляет обе куки
- **`POST /auth/logout`** — читает обе куки, вызывает `auth.Logout`

---

### 6. App Wiring

#### [MODIFY] [app.go](file:///Users/EV/Desktop/TechParkVK/clover-project/2026_1_VKernelTeam/internal/app/app.go)

- Убрать `InMemory`, создать `redis.Storage`
- Передать `redisStorage` и в `auth.New` как `TokenRevoker`, и как `RefreshStorage`
- Добавить `redisStorage.Close()` в `Stop()`

```go
redisStorage := redis.New(cfg.RedisAddr)

authService := auth.New(log, pgStorage, redisStorage, redisStorage, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)
```

---

### 7. InMemory blacklist — удаление

#### [DELETE] `internal/repository/blacklist/inmemory.go`
#### [DELETE] `internal/repository/blacklist/inmemory_test.go`

После переноса функциональности в Redis — пакет `blacklist` можно удалить.

---

### 8. Docker

#### [MODIFY] `deployments/docker-compose.yml`

Добавить сервис Redis:
```yaml
redis:
  image: redis:7-alpine
  restart: unless-stopped
  ports:
    - "6379:6379"
```

---

## Verification Plan

### Automated Tests

- Написать unit-тесты для `redis.Storage` с mock-пулом redigo (или тестовым Redis через `miniredis`)
- Обновить `auth_test.go`: добавить mock для `RefreshStorage`
- Проверить компиляцию: `go build ./...`
- Прогнать тесты: `go test ./...`

### Manual Verification

1. Запустить Redis локально: `redis-server` или через Docker
2. `POST /auth/login` → должны прийти две куки
3. Дождаться истечения access (или укорить TTL до 1 мин), затем `POST /auth/refresh` → новый access
4. `POST /auth/logout` → куки чистятся, refresh удаляется из Redis (`redis-cli GET refresh:<token>` → nil)
5. Попытка использовать старый access после logout → 401
