# Спецификация: Оценка обращений техподдержки (бэкенд)

**Дата**: 2026-04-25  
**Статус**: Реализовано

Фронтенд уже реализован и отправляет запрос `POST /api/v1/support/tickets/{id}/rate`.
Ниже — что добавлено на бэкенде.

---

## 1. Миграция БД

Добавлена колонка `rating` в таблицу `support_ticket`.

**`internal/repository/postgres/migrations/000015_add_rating_to_support_ticket.up.sql`**
```sql
ALTER TABLE support_ticket
    ADD COLUMN rating SMALLINT DEFAULT NULL
        CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5));
```

**`internal/repository/postgres/migrations/000015_add_rating_to_support_ticket.down.sql`**
```sql
ALTER TABLE support_ticket DROP COLUMN IF EXISTS rating;
```

---

## 2. Модель

Обновлена структура `SupportTicket` в `internal/domain/models/support_ticket.go`:

```go
type SupportTicket struct {
    ID          int64     `json:"id"`
    UserID      int64     `json:"user_id"`
    Category    string    `json:"category"`
    Status      string    `json:"status"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Rating      *int      `json:"rating"`       // nullable
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

DTO запроса (`internal/domain/dto/support_ticket.go`):

```go
type RateTicketRequest struct {
    Rating int `json:"rating"`
}
```

DTO ответа обновлён — добавлено поле `Rating *int`.

---

## 3. Репозиторий

Добавлен метод в `internal/repository/support_ticket/postgres.go`:

```go
SetRating(ctx context.Context, ticketID int64, rating int) error
```

Реализация:
```sql
UPDATE support_ticket
SET rating = $1, updated_at = NOW()
WHERE id = $2
```

Обновлены SQL-запросы `Create`, `GetByID`, `GetByUserID`, `GetAll` — добавлен `rating` в `SELECT` / `RETURNING` и `Scan`.

---

## 4. Сервис (usecase)

Добавлен метод в `internal/usecase/support_ticket/support_ticket.go`:

```go
func (s *SupportTicketService) RateTicket(ctx context.Context, userID, ticketID int64, rating int) (*dto.TicketResponse, error)
```

**Бизнес-правила:**
- `rating` должен быть от 1 до 5 — иначе вернуть `400 Bad Request`
- Тикет должен существовать — иначе `404 Not Found`
- Оценить может только автор тикета (`ticket.UserID == userID`) — иначе `403 Forbidden`
- Тикет должен быть в статусе `closed` — иначе `400 Bad Request` с сообщением «can only rate closed tickets»
- Если оценка уже выставлена (`ticket.Rating != nil`) — вернуть `400 Bad Request` с сообщением «ticket already rated»
- После успешной записи — вернуть обновлённый тикет

---

## 5. HTTP-хендлер

Добавлен в `internal/delivery/handlers/support_ticket.go`:

```go
func (h *SupportTicketHandlers) HandleRateTicket(w http.ResponseWriter, r *http.Request)
```

---

## 6. Роутинг

Добавлен в `setupRoutes()` (`internal/app/http/app.go`):

```go
a.router.Handle("POST "+prefix+"/support/tickets/{id}/rate", authMW(http.HandlerFunc(a.supportTicketHandlers.HandleRateTicket)))
```

---

## 7. API-контракт

### `POST /api/v1/support/tickets/{id}/rate`

**Авторизация:** Cookie JWT (user_id из токена)

**Запрос:**
```json
{
  "rating": 4
}
```

**Ответ (200):**
```json
{
  "id": 1,
  "user_id": 42,
  "category": "bug",
  "status": "closed",
  "title": "Не загружаются фото",
  "description": "При загрузке фото появляется ошибка 500",
  "rating": 4,
  "created_at": "2026-04-25T10:30:00Z",
  "updated_at": "2026-04-25T14:00:00Z"
}
```

**Ошибки:**

| Код | Когда |
|-----|-------|
| 400 | `rating` не в диапазоне 1–5 |
| 400 | Тикет не в статусе `closed` |
| 400 | Оценка уже выставлена |
| 401 | Нет токена / невалидный токен |
| 403 | Пользователь не является автором тикета |
| 404 | Тикет не найден |

---

## 8. Фронтенд-контракт (уже реализовано)

Фронтенд вызывает:
```typescript
supportApi.rateTicket(ticketId, rating)
// → POST /api/v1/support/tickets/{id}/rate  body: { rating: N }
```

- Поле `rating` в `SupportTicket` типа `number | null`
- Шаблон `ticket-detail.hbs` показывает звёзды только при `status === "closed"`
- Если `ticket.rating` не null — отображает readonly-звёзды с текстом «Спасибо за обратную связь!»
- Если null — интерактивные звёзды с hover-эффектом, клик отправляет запрос

---

## 9. Чеклист реализации

- [x] Миграция: добавить колонку `rating SMALLINT` с `CHECK (1..5)`
- [x] Модель: добавить поле `Rating *int` в структуру
- [x] Репозиторий: метод `SetRating`, обновить SELECT-запросы
- [x] Сервис: метод `RateTicket` с валидацией
- [x] Хендлер: `HandleRateTicket` — парсинг body, вызов сервиса, формирование ответа
- [x] Роут: `POST /api/v1/support/tickets/{id}/rate`
- [x] Формат ошибок: через существующий `responser` из `pkg/responser`
