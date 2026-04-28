# Auth & Users Service

Микросервис аутентификации и управления пользователями.

## Запуск

```bash
# Из корня репозитория
go build -o auth-service ./services/auth/cmd/server
CONFIG_PATH=services/auth/config/local.json ./auth-service
```

Сервис поднимает два сервера:
- **HTTP** `:8001` — для фронтенда (регистрация, логин, профиль)
- **gRPC** `:9001` — для других микросервисов (валидация токенов, данные пользователей)

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `DATABASE_DSN` | PostgreSQL DSN (своя БД auth) |
| `REDIS_ADDR` | Адрес Redis |
| `TOKEN_SECRET` | Секрет для подписи JWT |
| `S3_ENDPOINT_URL` | Эндпоинт S3 |
| `S3_REGION_NAME` | Регион S3 |
| `S3_BUCKET_NAME` | Бакет S3 |
| `S3_ACCESS_KEY_ID` | S3 ключ доступа |
| `S3_SECRET_ACCESS_KEY` | S3 секретный ключ |

## HTTP API

### Публичные

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/auth/register` | Регистрация |
| POST | `/api/v1/auth/login` | Авторизация |
| POST | `/api/v1/auth/refresh` | Обновление токена |
| GET | `/api/v1/users/{id}` | Публичный профиль |

### Требуют авторизации (JWT в cookie)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/auth/logout` | Выход |
| GET | `/api/v1/profile` | Свой профиль |
| PATCH | `/api/v1/profile` | Обновление профиля |
| POST | `/api/v1/profile/avatar` | Загрузка аватара |

## gRPC API (для Catalog, Commerce, Support)

Контракт: `proto/auth/v1/auth.proto`

Сгенерированный код: `proto/gen/auth/v1/`

### Подключение из другого сервиса

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
)

conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := authv1.NewAuthServiceClient(conn)
```

### Методы

#### `ValidateToken` — проверка JWT

Вызывай из auth middleware своего сервиса. Каждый входящий HTTP-запрос с cookie `token` проверяется через этот метод.

```go
resp, err := client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
    Token: "eyJhbGciOiJIUzI1NiIs...",
})
// resp.UserId — ID пользователя
// resp.Role   — "user", "support" или "admin"
```

**gRPC коды ошибок:**
- `InvalidArgument` — пустой токен
- `Unauthenticated` — невалидный или просроченный токен

#### `GetUser` — данные пользователя по ID

Используй когда нужно показать имя/аватар пользователя (продавец в объявлении, участник чата).

```go
resp, err := client.GetUser(ctx, &authv1.GetUserRequest{
    UserId: 42,
})
// resp.FirstName, resp.Email, resp.AvatarPath, resp.Rating, resp.Role
```

**gRPC коды ошибок:**
- `InvalidArgument` — user_id == 0
- `NotFound` — пользователь не найден

#### `GetUsersByIDs` — пакетное получение

Используй вместо множества `GetUser` когда нужно получить несколько пользователей за раз (список чатов, список продавцов).

```go
resp, err := client.GetUsersByIDs(ctx, &authv1.GetUsersByIDsRequest{
    UserIds: []int64{1, 2, 3, 42},
})
// resp.Users — массив UserResponse
// Несуществующие ID пропускаются (не ломают запрос)
```

#### `CheckRole` — проверка роли

Используй в Support-сервисе для защиты админских эндпоинтов.

```go
resp, err := client.CheckRole(ctx, &authv1.CheckRoleRequest{
    UserId:       42,
    RequiredRole: "support",
})
// resp.Allowed    — true/false
// resp.ActualRole — "user", "support", "admin"
// admin имеет доступ ко всем ролям
```

**gRPC коды ошибок:**
- `InvalidArgument` — пустой user_id или required_role
- `NotFound` — пользователь не найден

## БД (своя, отдельная от монолита)

Миграции: `services/auth/migrations/`

Таблицы:
- `user` — профили, роли, рейтинг
- `refresh_token` — refresh-токены

## Структура

```
services/auth/
├── cmd/server/main.go               # Точка входа (HTTP + gRPC)
├── config/local.json                 # Конфиг (порты, TTL)
├── internal/
│   ├── config/                       # Парсинг конфига
│   ├── delivery/
│   │   ├── grpc/server.go            # gRPC сервер (ValidateToken, GetUser...)
│   │   └── handlers/                 # HTTP handlers (auth, profile)
│   ├── domain/
│   │   ├── dto/                      # Request/Response структуры
│   │   └── models/                   # User модель
│   ├── repository/
│   │   ├── blacklist/                # JWT blacklist (Redis)
│   │   ├── postgres/                 # PostgreSQL клиент
│   │   ├── redis/                    # Redis клиент
│   │   ├── refresh/                  # Refresh-токены (Redis)
│   │   ├── s3/                       # S3 клиент (аватары)
│   │   └── user/                     # CRUD пользователей (PostgreSQL)
│   └── usecase/auth/                 # Бизнес-логика
├── migrations/                       # SQL миграции
└── pkg/jwt/                          # JWT генерация/парсинг
```

Shared-пакеты (`pkg/validator`, `pkg/responser`, `pkg/sanitizer`, `pkg/http/middleware`) импортируются из корня репозитория — единый `go.mod`.

## Отличия от монолита

1. **UserByID** — убраны JOIN на `product`, `favorite`, `cart_item`, `message` (эти таблицы теперь в Catalog и Commerce). Счётчики `ads_count`, `cart_count` и т.д. не заполняются на уровне Auth.

2. **Handlers** — `AuthHandlers` принимает только интерфейс `Auth`, а не все сервисы (Ads, Cart, Chat). Каждый сервис знает только о своих зависимостях.

3. **Конфиг** — только auth-related настройки (убраны `search`, `views`).
