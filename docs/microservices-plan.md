# План декомпозиции монолита Clover на микросервисы

> Дата: 2026-04-28
> Команда: VKernelTeam
> Разработчики: Егор Воробьёв, Николай

---

## 1. Обзор

Текущий монолит разбивается на **4 микросервиса** по доменным границам.
Каждый сервис получает собственную базу данных (Database per Service),
общение между сервисами — **gRPC** (синхронно) и **Apache Kafka** (асинхронные события).

### Распределение сервисов

| Сервис | Ответственный | Фаза |
|--------|---------------|-------|
| **Auth & Users** | Егор | 1 |
| **Catalog** | Николай | 1 |
| **Support** | Егор | 2 |
| **Commerce** (Cart + Orders + Chat) | Николай | 2 |

**Логика распределения:** Auth и Support тесно связаны через систему ролей
(admin/support) — Егор владеет обоими и не зависит от другого разработчика
для интеграции проверки прав. Catalog и Commerce связаны через данные о товарах
(корзина, заказы, чаты) — Николай владеет обоими.

### Архитектура верхнего уровня

```
                 ┌────────────────┐
                 │   API Gateway  │
                 │  (nginx/traefik)│
                 └───────┬────────┘
        ┌────────┬───────┼────────┬──────────┐
        ▼        ▼       ▼        ▼          ▼
   ┌────────┐ ┌───────┐ ┌────────┐ ┌────────┐
   │  Auth  │ │Catalog│ │Commerce│ │Support │
   │ (Егор) │ │(Никол)│ │(Никол) │ │ (Егор) │
   └───┬────┘ └───┬───┘ └───┬────┘ └───┬────┘
       │          │         │           │
  ─────┴──────────┴─────────┴───────────┴─────
       gRPC (синхронно) + Kafka (события)
```

---

## 2. Описание сервисов

### 2.1 Auth & Users (Егор)

**Зона ответственности:** аутентификация, управление пользователями, профили, роли.

**Таблицы (своя БД):**
- `user` — профили, роли, рейтинг
- `refresh_token` — refresh-токены

**Инфраструктура:**
- Redis — blacklist отозванных JWT, кэш сессий
- S3 — хранение аватаров

**HTTP API (внешний):**

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/auth/register` | Регистрация |
| POST | `/api/v1/auth/login` | Авторизация |
| POST | `/api/v1/auth/logout` | Выход |
| POST | `/api/v1/auth/refresh` | Обновление токена |
| GET | `/api/v1/profile` | Свой профиль |
| PATCH | `/api/v1/profile` | Обновление профиля |
| POST | `/api/v1/profile/avatar` | Загрузка аватара |
| GET | `/api/v1/users/{id}` | Публичный профиль |

**gRPC API (внутренний, для других сервисов):**

```protobuf
service AuthService {
  // Валидация JWT токена, возвращает user_id и роль
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);

  // Получение данных пользователя по ID
  rpc GetUser(GetUserRequest) returns (UserResponse);

  // Пакетное получение пользователей (для чатов, объявлений)
  rpc GetUsersByIDs(GetUsersByIDsRequest) returns (GetUsersByIDsResponse);

  // Проверка роли пользователя
  rpc CheckRole(CheckRoleRequest) returns (CheckRoleResponse);
}
```

**Kafka события (публикует в топик `clover.auth.user-events`):**
- `user.deleted` — пользователь удалён (для очистки данных в других сервисах)
- `user.updated` — профиль обновлён (для инвалидации кэшей)

---

### 2.2 Catalog (Николай)

**Зона ответственности:** объявления, поиск, категории, характеристики, фото, избранное, просмотры.

**Таблицы (своя БД):**
- `product` — объявления
- `product_image` — фотографии
- `category` — категории
- `category_characteristic` — характеристики категорий
- `product_characteristic` — характеристики товара
- `product_custom_characteristic` — пользовательские характеристики
- `favorite` — избранное
- `product_view` — просмотры
- `product_price_history` — история цен
- `product_status_history` — история статусов

**Инфраструктура:**
- Redis — дедупликация просмотров, Redis Streams (view pipeline), кэш
- S3 — хранение фото товаров

**HTTP API (внешний):**

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/ads` | Список объявлений |
| GET | `/api/v1/ads/search` | Поиск |
| GET | `/api/v1/ads/{id}` | Детали объявления |
| POST | `/api/v1/ads` | Создание (auth) |
| PUT | `/api/v1/ads/{id}` | Обновление (auth) |
| DELETE | `/api/v1/ads/{id}` | Удаление (auth) |
| POST | `/api/v1/ads/{id}/close` | Закрытие (auth) |
| POST | `/api/v1/ads/{id}/favorite` | В избранное (auth) |
| DELETE | `/api/v1/ads/{id}/favorite` | Из избранного (auth) |
| GET | `/api/v1/profile/favorites` | Мои избранные (auth) |
| GET | `/api/v1/users/{id}/ads` | Объявления пользователя |
| GET | `/api/v1/ads/{id}/view` | Записать просмотр |
| GET | `/api/v1/categories/{id}/characteristics` | Характеристики категории |

**gRPC API (внутренний):**

```protobuf
service CatalogService {
  // Получение объявления по ID (для Commerce — цена, статус)
  rpc GetAd(GetAdRequest) returns (AdResponse);

  // Пакетное получение объявлений
  rpc GetAdsByIDs(GetAdsByIDsRequest) returns (GetAdsByIDsResponse);

  // Проверка статуса (для корзины — можно ли добавить)
  rpc CheckAdStatus(CheckAdStatusRequest) returns (CheckAdStatusResponse);

  // Смена статуса товара (при покупке через Commerce)
  rpc UpdateAdStatus(UpdateAdStatusRequest) returns (UpdateAdStatusResponse);
}
```

**gRPC клиенты (вызывает):**
- `AuthService.ValidateToken` — проверка токена в auth middleware

**Kafka события (публикует в топик `clover.catalog.ad-events`):**
- `ad.sold` — товар продан
- `ad.deleted` — товар удалён (для очистки корзин)

---

### 2.3 Support (Егор)

**Зона ответственности:** тикеты поддержки, переписка в тикетах, статистика.

**Связь с Auth:** Support тесно завязан на систему ролей (admin/support) из Auth.
Поскольку Егор владеет обоими сервисами, интеграция проверки ролей через gRPC
реализуется без межкомандных зависимостей.

**Таблицы (своя БД):**
- `support_ticket` — тикеты
- `support_message` — сообщения

**HTTP API (внешний):**

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/support/tickets` | Создать тикет |
| GET | `/api/v1/support/tickets` | Мои тикеты |
| GET | `/api/v1/support/tickets/{id}` | Детали тикета |
| PUT | `/api/v1/support/tickets/{id}` | Обновить тикет |
| POST | `/api/v1/support/tickets/{id}/rate` | Оценить |
| POST | `/api/v1/support/tickets/{id}/messages` | Отправить сообщение |
| GET | `/api/v1/support/tickets/{id}/messages` | Сообщения тикета |
| PATCH | `/api/v1/support/tickets/{id}/status` | Сменить статус (admin) |
| GET | `/api/v1/support/tickets/all` | Все тикеты (admin) |
| GET | `/api/v1/support/tickets/stats` | Статистика (admin) |

**gRPC клиенты (вызывает):**
- `AuthService.ValidateToken` — auth middleware
- `AuthService.CheckRole` — проверка роли support/admin

**Kafka подписки (слушает):**
- `user.deleted` — анонимизация тикетов удалённого пользователя

---

### 2.4 Commerce (Николай)

**Зона ответственности:** корзина, заказы, чаты покупатель-продавец.

**Связь с Catalog:** Commerce напрямую работает с данными о товарах (проверка статуса,
цена при покупке, информация для чатов). Поскольку Николай владеет обоими сервисами,
gRPC интеграция Catalog ↔ Commerce реализуется без межкомандных зависимостей.

**Таблицы (своя БД):**
- `cart_item` — корзина
- `order` — заказы
- `order_item` — позиции заказа
- `chat` — чаты
- `message` — сообщения

**HTTP API (внешний):**

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/cart` | Содержимое корзины |
| POST | `/api/v1/cart` | Добавить в корзину |
| DELETE | `/api/v1/cart/{id}` | Убрать из корзины |
| POST | `/api/v1/ads/{id}/order` | Создать заказ/чат |
| POST | `/api/v1/chats/{id}/confirm` | Подтвердить покупку |
| GET | `/api/v1/chats` | Список чатов |
| GET | `/api/v1/chats/{id}` | Детали чата |

**gRPC клиенты (вызывает):**
- `AuthService.ValidateToken` — auth middleware
- `AuthService.GetUser` / `GetUsersByIDs` — данные покупателя/продавца в чатах
- `CatalogService.GetAd` — информация о товаре
- `CatalogService.CheckAdStatus` — валидация при добавлении в корзину
- `CatalogService.UpdateAdStatus` — смена статуса при покупке

**Kafka подписки (слушает):**
- `ad.deleted` — удалить товар из всех корзин

---

## 3. План по дням (вт 28.04 — сб 02.05)

> **Дедлайн: суббота 02.05.2026**
> Сжатый план на 5 дней. Фокус на работающие сервисы, тесты — по остаточному принципу.

---

### Вторник 28.04 — Подготовка (совместно, 1 день)

**Утро (совместно):**

| # | Задача | Егор | Николай |
|---|--------|:----:|:-------:|
| 0.1 | Создать monorepo структуру с `go.work`, директории сервисов | + | + |
| 0.2 | Написать `.proto` файлы: `auth.proto`, `catalog.proto` + кодогенерация | + | |
| 0.3 | Выделить shared-пакеты в `pkg/shared/` (jwt, validator, responser, sanitizer, middleware) | | + |

**Вечер (параллельно):**

| # | Задача | Егор | Николай |
|---|--------|:----:|:-------:|
| 0.4 | Kafka: docker-compose (KRaft) + shared Go-клиент `pkg/shared/kafka/` | + | |
| 0.5 | Docker-compose: отдельные БД на сервис, Redis | | + |
| 0.6 | Согласовать топики Kafka и формат событий | + | + |

**Контрольная точка вечер вт:** monorepo собирается, `docker-compose up` поднимает все БД + Redis + Kafka.

**Структура monorepo:**

```
clover/
├── go.work
├── proto/
│   ├── auth/v1/auth.proto
│   └── catalog/v1/catalog.proto
├── pkg/
│   └── shared/           # общие пакеты
│       ├── jwt/
│       ├── validator/
│       ├── responser/
│       ├── sanitizer/
│       ├── middleware/
│       └── kafka/        # producer/consumer обёртки (Егор)
├── services/
│   ├── auth/             # Егор
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   ├── migrations/
│   │   └── go.mod
│   ├── catalog/          # Николай
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   ├── migrations/
│   │   └── go.mod
│   ├── support/          # Егор
│   │   └── ...
│   └── commerce/         # Николай
│       └── ...
├── deployments/
│   └── docker-compose.yaml
└── Makefile
```

---

### Среда 29.04 — Auth & Catalog: ядро (параллельно)

#### Егор: Auth & Users

| Время | Задачи |
|-------|--------|
| **Утро** | Создать `services/auth/`, перенести миграции (user, refresh_token). Перенести repository: user, refresh, blacklist. |
| **День** | Перенести usecase auth. Перенести handlers (register, login, logout, refresh, profile). |
| **Вечер** | Реализовать gRPC сервер (ValidateToken, GetUser, GetUsersByIDs, CheckRole). Проверить что сервис стартует и отвечает. |

#### Николай: Catalog

| Время | Задачи |
|-------|--------|
| **Утро** | Создать `services/catalog/`, перенести миграции (product, category, images, characteristics, favorite, product_view). |
| **День** | Перенести repository: ad, view, view_cache, view_stream. Перенести usecase ads и views. |
| **Вечер** | Перенести handlers. Проверить что сервис стартует и отвечает на HTTP. |

**Контрольная точка вечер ср:** Auth выдаёт токены, gRPC работает. Catalog отдаёт объявления. Сервисы запускаются независимо.

---

### Четверг 30.04 — Auth & Catalog: интеграция + gRPC (параллельно)

#### Егор: Auth — завершение

| Время | Задачи |
|-------|--------|
| **Утро** | Подключить S3 для аватаров. Auth middleware через gRPC (шаблон для всех сервисов). |
| **День** | Kafka producer: публикация user.deleted, user.updated в `clover.auth.user-events`. |
| **Вечер** | Начать `services/support/`: миграции, repository, usecase. |

#### Николай: Catalog — завершение

| Время | Задачи |
|-------|--------|
| **Утро** | gRPC сервер Catalog (GetAd, GetAdsByIDs, CheckAdStatus, UpdateAdStatus). Интегрировать Auth gRPC клиент для middleware. |
| **День** | Перенести search (translit, synonyms). Подключить S3 для фото. |
| **Вечер** | Kafka producer: ad.sold, ad.deleted в `clover.catalog.ad-events`. Начать `services/commerce/`: миграции, repository. |

**Контрольная точка вечер чт:** Auth + Catalog полностью работают, Catalog проверяет токены через gRPC Auth. Kafka продюсеры настроены. Заготовки Support и Commerce созданы.

---

### Пятница 01.05 — Support & Commerce (параллельно)

#### Егор: Support

| Время | Задачи |
|-------|--------|
| **Утро** | Перенести handlers support. Интегрировать Auth gRPC клиент (токены + CheckRole). |
| **День** | Kafka consumer: подписка на `clover.auth.user-events` → `user.deleted` (анонимизация). |
| **Вечер** | Smoke-тест полного флоу Auth → Support. Фиксы. |

#### Николай: Commerce

| Время | Задачи |
|-------|--------|
| **Утро** | Перенести usecase cart и chat. Перенести handlers. |
| **День** | gRPC клиенты: Auth (токены, пользователи) + Catalog (статус товара). |
| **Вечер** | Kafka consumer: `clover.catalog.ad-events` → `ad.deleted` (очистка корзин). Smoke-тест. |

**Контрольная точка вечер пт:** все 4 сервиса работают. Основные API-эндпоинты функционируют.

---

### Суббота 02.05 — Интеграция и финализация (совместно)

| Время | Задача | Егор | Николай |
|-------|--------|:----:|:-------:|
| **Утро** | API Gateway (nginx): маршрутизация по path prefix | + | |
| **Утро** | Финальный docker-compose: 4 сервиса + БД + Redis + Kafka | | + |
| **День** | E2E тест: регистрация → объявление → корзина → чат → поддержка | + | + |
| **День** | Фиксы интеграционных багов | + | + |
| **Вечер** | Health checks, graceful shutdown | + | + |
| **Вечер** | Обновить Swagger (per-service) | + | + |

**Финальная контрольная точка:** полный flow работает через API Gateway, все сервисы поднимаются одним `docker-compose up`.

---

## 4. Что откладывается из-за сжатых сроков

| Задача | Приоритет | Когда |
|--------|-----------|-------|
| Saga pattern для покупки (Commerce → Catalog) | Средний | Заменяем на прямой gRPC вызов, saga — после дедлайна |
| Outbox pattern для Kafka | Низкий | После дедлайна |
| Unit-тесты с mock gRPC | Средний | После дедлайна |
| Интеграционные тесты (testcontainers) | Средний | После дедлайна |
| CI/CD pipeline | Низкий | После дедлайна |
| Нагрузочное тестирование | Низкий | После дедлайна |

---

## 5. Правила работы

### Git-flow
- Ветка `microservices/base` — от `dev`, содержит monorepo структуру (вторник)
- `microservices/auth` — Егор
- `microservices/catalog` — Николай
- `microservices/support` — Егор
- `microservices/commerce` — Николай
- Merge в `microservices/base` по мере готовности, быстрый review

### Контракты
- `.proto` файлы фиксируются во вторник, изменения — только по согласованию
- `pkg/shared/` — быстрый review, не блокирующий

### Коммуникация
- Синк утром и вечером (15 мин): что сделано, что блокирует
- Блокеры — решать сразу, не откладывать

---

## 6. Риски

| Риск | Митигация |
|------|-----------|
| Не успеваем за 5 дней | Приоритет: Auth + Catalog (ср-чт), Support + Commerce можно в упрощённом виде |
| gRPC интеграция ломается | Фиксируем proto во вторник, не меняем без синка |
| Kafka не успевают подключить | Kafka consumer/producer можно добавить последними, сервисы работают и без событий |
| Баги при интеграции в субботу | Закладываем весь день субботы только на интеграцию и фиксы |

---

## 6. Kafka: инфраструктура и топики

### Ответственный за настройку: Егор (фаза 0)

### Docker-compose (KRaft mode, без Zookeeper)

```yaml
kafka:
  image: confluentinc/cp-kafka:7.6.0
  ports:
    - "9092:9092"
  environment:
    KAFKA_NODE_ID: 1
    KAFKA_PROCESS_ROLES: broker,controller
    KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
    KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
    KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
    KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT
    KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
    KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
    KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    CLUSTER_ID: "clover-local-cluster"
  volumes:
    - kafka_data:/var/lib/kafka/data
```

### Топики

| Топик | Продюсер | Консьюмеры | Partition key |
|-------|----------|------------|---------------|
| `clover.auth.user-events` | Auth (Егор) | Support (Егор), Catalog (Николай) | `user_id` |
| `clover.catalog.ad-events` | Catalog (Николай) | Commerce (Николай) | `ad_id` |

### Формат сообщений

```json
{
  "event_type": "user.deleted",
  "event_id": "uuid",
  "timestamp": "2026-04-28T12:00:00Z",
  "payload": { ... }
}
```

### Shared Go-клиент (`pkg/shared/kafka/`)

```
pkg/shared/kafka/
├── producer.go    # обёртка над segmentio/kafka-go
├── consumer.go    # consumer group с graceful shutdown
├── config.go      # конфигурация (brokers, topics, group ID)
└── events.go      # общие типы: UserEvent, AdEvent
```

### Карта событий по разработчикам

```
Егор:
  Auth  ──produce──► clover.auth.user-events
  Support ◄─consume── clover.auth.user-events  (свой же топик)

Николай:
  Catalog ──produce──► clover.catalog.ad-events
  Commerce ◄─consume── clover.catalog.ad-events  (свой же топик)
  Catalog  ◄─consume── clover.auth.user-events   (кросс-команда: инвалидация кэша)
```

> Кросс-командная зависимость минимальна: только Catalog (Николай)
> слушает топик Auth (Егор) для инвалидации кэша данных продавцов.

---

## 8. Итоговый таймлайн

```
Вт 28.04    Подготовка: monorepo, proto, docker, Kafka, shared-пакеты
            ✓ docker-compose up поднимает инфраструктуру
            ┌───────────────────────────────────────────────┐
Ср 29.04    │ Егор: Auth (ядро)      │ Николай: Catalog (ядро) │
            ├───────────────────────────────────────────────┤
Чт 30.04    │ Егор: Auth (gRPC,Kafka)│ Николай: Catalog (gRPC) │
            │       + начало Support │        + начало Commerce│
            └───────────────────────────────────────────────┘
            ✓ Auth + Catalog работают, gRPC связаны
            ┌───────────────────────────────────────────────┐
Пт 01.05    │ Егор: Support          │ Николай: Commerce       │
            └───────────────────────────────────────────────┘
            ✓ Все 4 сервиса работают
Сб 02.05    Интеграция: API Gateway, docker-compose, E2E, фиксы
            ✓ Готово
```

**Общая длительность: 5 дней (вт–сб)**
