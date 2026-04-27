# Ad Search Design

## Overview

Полнотекстовый поиск по объявлениям маркетплейса Clover с поддержкой триграмм, транслитерации, смены раскладки и защитой БД от перегрузки.

## Требования

- Поиск по `title` + `description`
- Нечёткий поиск через триграммы (pg_trgm)
- Транслитерация: поиск английских слов русскими буквами и наоборот (MacBook → макбук)
- Смена раскладки: поиск при забытом переключении QWERTY↔ЙЦУКЕН (vfr,er → макбук)
- Защита БД от перегрузки
- Top-N результатов (без пагинации)
- Конфигурируемые параметры поиска

## Архитектура

```
Handler (GET /api/v1/ads/search?query=...)
   |
   v
Usecase (валидация min_query_length, вызов транслитерации)
   |
   v
Translit (internal/pkg/translit/) -- генерация вариантов:
   - фонетическая транслитерация (MacBook → макбук)
   - смена раскладки QWERTY↔ЙЦУКЕН (vfr,er → макбук)
   |
   v
Repository (SQL с pg_trgm word_similarity, SET LOCAL threshold, CTE + LIMIT)
   |
   v
PostgreSQL (GIN-индексы, pg_trgm)
```

## 1. Схема БД и индексы

### Миграция

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_product_title_trgm ON product USING GIN (title gin_trgm_ops);
CREATE INDEX idx_product_description_trgm ON product USING GIN (description gin_trgm_ops);
```

### SQL-запрос поиска

Двухфазный подход через CTE: сначала находим ID подходящих товаров с ранжированием,
затем подтягиваем связанные данные (фото, просмотры, избранное). Это исключает
декартово произведение при JOIN множественных связей.

Для `title` (короткие строки) используется `similarity` / `%`.
Для `description` (длинные тексты) используется `word_similarity` / `<%` — он ищет
схожесть запроса с отдельными словами внутри текста, а не со всей строкой целиком.

Поиск ведётся по трём вариантам запроса: оригинал ($1), транслитерация ($2), смена раскладки ($3).

```sql
BEGIN;

SET LOCAL pg_trgm.similarity_threshold = $4;
SET LOCAL pg_trgm.word_similarity_threshold = $5;

WITH matched AS (
    SELECT p.id,
           greatest(
               similarity(p.title, $1),
               similarity(p.title, $2),
               similarity(p.title, $3),
               word_similarity($1, p.description),
               word_similarity($2, p.description),
               word_similarity($3, p.description)
           ) AS rank
    FROM product p
    WHERE p.deleted_at IS NULL
      AND p.status = 'active'
      AND (
          p.title % $1 OR p.title % $2 OR p.title % $3
          OR $1 <% p.description OR $2 <% p.description OR $3 <% p.description
      )
    ORDER BY rank DESC
    LIMIT $6
)
SELECT p.id, p.seller_id, p.category_id, p.title, p.description,
       p.price, p.status, p.location, p.created_at, p.updated_at,
       array_agg(DISTINCT pi.file_path) FILTER (WHERE pi.file_path IS NOT NULL) AS photos,
       count(DISTINCT pv.id) AS views_count,
       count(DISTINCT f.id) AS favorites_count,
       m.rank
FROM matched m
JOIN product p ON p.id = m.id
LEFT JOIN product_image pi ON pi.product_id = p.id
LEFT JOIN product_view pv ON pv.product_id = p.id
LEFT JOIN favorite f ON f.product_id = p.id
GROUP BY p.id, m.rank
ORDER BY m.rank DESC;

COMMIT;
```

- `$1` -- оригинальный запрос
- `$2` -- фонетическая транслитерация
- `$3` -- смена раскладки
- `$4` -- similarity_threshold из конфигурации
- `$5` -- word_similarity_threshold из конфигурации
- `$6` -- max_results из конфигурации

### Почему SET LOCAL, а не SET

`SET LOCAL` действует только в рамках текущей транзакции. При использовании pgxpool
обычный `SET` сохраняется на коннекте и может сломать логику других запросов,
переиспользующих то же соединение. `SET LOCAL` безопасен — после COMMIT/ROLLBACK
значение автоматически сбрасывается.

## 2. Транслитерация и смена раскладки в Go

### Пакет: `internal/pkg/translit/translit.go`

Две функции преобразования:

#### 2.1 Фонетическая транслитерация

Таблица маппинга:

```
а↔a   б↔b   в↔v   г↔g   д↔d   е↔e   ж↔zh   з↔z   и↔i   й↔y
к↔k   л↔l   м↔m   н↔n   о↔o   п↔p   р↔r    с↔s   т↔t   у↔u
ф↔f   х↔kh  ц↔ts  ч↔ch  ш↔sh  щ↔shch       ы↔y   ь↔    ъ↔
э↔e   ю↔yu  я↔ya
```

#### 2.2 Смена раскладки QWERTY↔ЙЦУКЕН

Позиционный маппинг клавиш:

```
q↔й  w↔ц  e↔у  r↔к  t↔е  y↔н  u↔г  i↔ш  o↔щ  p↔з
a↔ф  s↔ы  d↔в  f↔а  g↔п  h↔р  j↔о  k↔л  l↔д
z↔я  x↔ч  c↔с  v↔м  b↔и  n↔т  m↔ь
```

#### 2.3 Логика генерации вариантов

Функция `GenerateVariants(query string) (original, translit, layout string)`:

1. Определить, содержит ли строка кириллицу, латиницу или смешанный текст
2. Сгенерировать фонетическую транслитерацию (ru→en или en→ru)
3. Сгенерировать вариант со сменой раскладки (QWERTY→ЙЦУКЕН или наоборот)
4. Вернуть три варианта для поиска

Если вариант совпадает с оригиналом, передаётся как есть (дубли не вредят производительности SQL).

## 3. Защита БД от перегрузки

### Три уровня:

1. **Жёсткий LIMIT в CTE** -- поиск ID ограничен `max_results` (default: 50), JOIN-ы только к найденным записям
2. **Минимальная длина запроса** -- отклоняем запросы короче `min_query_length` (default: 2) на уровне usecase
3. **Пороги similarity через SET LOCAL** -- `similarity_threshold` и `word_similarity_threshold` устанавливаются в рамках транзакции, PostgreSQL отсекает нерелевантные строки на уровне индекса

## 4. Конфигурация

### Добавляется в `config/local.json`:

```json
{
  "search": {
    "max_results": 50,
    "min_query_length": 2,
    "similarity_threshold": 0.1,
    "word_similarity_threshold": 0.1
  }
}
```

### Структура в `internal/config/config.go`:

```go
type SearchConfig struct {
    MaxResults              int     `json:"max_results"`
    MinQueryLength          int     `json:"min_query_length"`
    SimilarityThreshold     float64 `json:"similarity_threshold"`
    WordSimilarityThreshold float64 `json:"word_similarity_threshold"`
}
```

## 5. Новые и изменяемые файлы

### Новые файлы:
- `internal/pkg/translit/translit.go` -- пакет транслитерации и смены раскладки
- `internal/pkg/translit/translit_test.go` -- тесты
- Новая миграция -- `pg_trgm` + GIN-индексы

### Изменяемые файлы:
- `internal/config/config.go` -- структура `SearchConfig`
- `internal/repository/ad/postgres.go` -- метод `SearchAds`
- `internal/usecase/ads/ads.go` -- метод `SearchAds` с валидацией
- `internal/delivery/handlers/ads.go` -- хендлер поиска
- `internal/app/http/app.go` -- регистрация маршрута

## 6. Endpoint

- `GET /api/v1/ads/search?query=макбук` -- публичный, без авторизации

## 7. Масштабируемость

Текущий подход работает до ~500K записей. При росте:
- Добавить Redis-кэширование популярных запросов
- Добавить rate limiting на middleware уровне
- Мигрировать на внешний поисковый движок (Meilisearch/Elasticsearch)
