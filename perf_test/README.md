# ДЗ 4 — Оптимизация работы СУБД

Маркетплейс Clover. Основная сущность — объявление (`product`, оно же *ad*).
Тестируем два эндпоинта catalog-сервиса: `POST /api/v1/ads` (создание) и
`GET /api/v1/ads` (листинг). Дополнительно — `GET /api/v1/ads/{id}` и
`GET /api/v1/ads/search`.

## Структура

```
perf_test/
├── README.md                ← этот файл
├── init.sql                 ← снимок DDL базы до оптимизаций (pg_dump --schema-only)
├── cmd/loader/main.go       ← кастомный нагрузочный инструмент на Go
├── bin/loader               ← собранный бинарь (git-ignore не настроен, можно собрать заново)
├── scripts/
│   ├── up.sh                ← поднимает docker-compose стек + ждёт catalog
│   ├── clean_db.sh          ← TRUNCATE всех зависимых от product таблиц
│   ├── run_baseline.sh      ← полный цикл: create 100k → read by-id → list → search
│   └── run_reads.sh         ← только read-сценарии (после набивки)
├── migrations/
│   └── iter1_indexes.sql    ← индексы, добавленные в первой итерации
├── explain/                 ← EXPLAIN ANALYZE для запросов до/после
└── results/                 ← JSON + log отчёты вегеты-стайл по каждому прогону
```

## Воспроизведение

```bash
# 0. Поднять стек (DB, redis, kafka, auth, catalog)
./perf_test/scripts/up.sh

# 1. Прогнать baseline (чистая БД → 100k create → reads)
./perf_test/scripts/run_baseline.sh baseline

# 2. Накатить индексы первой итерации
docker exec -e PGPASSWORD=qwerty -i postgres \
  psql -U postgres -d clover < perf_test/migrations/iter1_indexes.sql

# 3. Прогнать только reads на уже наполненной БД
./perf_test/scripts/run_reads.sh iter1_indexes

# 4. Применить код-оптимизацию (LIMIT/OFFSET в GetAllAds) и пересобрать catalog
docker compose --env-file ./.env -f deployments/docker-compose.yaml \
  up -d --build catalog

# 5. Прогнать reads ещё раз
./perf_test/scripts/run_reads.sh iter2_pagination
```

Все три действия — `up.sh` → `run_baseline.sh` → анализ JSON в `results/`.

## Инструмент

Использовал собственный Go-loader (`perf_test/cmd/loader/main.go`):

- Поддерживает auth (регистрация + login через JSON, кладёт `token` и `csrf_token`
  куки + заголовок `X-CSRF-Token`, как требует CSRF middleware каталога).
- `create` — POST `/api/v1/ads` (multipart/form-data, поле `data` с JSON).
- `read --mode by-id|list|search` — GET-ы.
- Считает RPS, p50/p95/p99, max, разбивку по 2xx/4xx/5xx.
- Пишет отчёт в JSON для последующей агрегации.

Готовые инструменты типа `vegeta` / `wrk` тоже подошли бы для read-эндпоинтов, но
для POST `/api/v1/ads` (multipart + auth-cookie + CSRF-header) проще иметь один
бинарь, который сам логинится и сам конструирует тело.

## Цель и сценарии

| # | Сценарий | Метод | Эндпоинт | Сколько | Concurrency |
|---|----------|-------|----------|---------|-------------|
| 1 | Create | POST | `/api/v1/ads` | 100 000 | 100 |
| 2 | Read by id | GET | `/api/v1/ads/{id}` | 10 000 | 100 |
| 3 | List | GET | `/api/v1/ads` | 100 | 10 |
| 4 | Search | GET | `/api/v1/ads/search?query=Объявление` | 1 000 | 50 |

Тестовое окружение: MacBook (darwin/amd64), docker-compose,
`postgres:17-alpine` с дефолтными `work_mem=4MB`, `shared_buffers=128MB`.
Catalog/auth — образы Go 1.26.

---

## Итерация 0 — baseline

Состояние БД: пусто, 58 пользователей-сидов из миграций. Делаем 100k POST
`/api/v1/ads`, затем 10k случайных GET `/api/v1/ads/{id}` и 100 GET `/api/v1/ads`.

### Результаты

| Сценарий | total | ok | err | wall | RPS | avg | p50 | p95 | p99 | max |
|----------|-------|----|----|------|-----|-----|-----|-----|-----|-----|
| **CREATE** 100k | 100 000 | 100 000 | 0 | 12.48 s | **8 016** | 12 ms | 12 ms | 16 ms | 25 ms | 77 ms |
| **READ by-id** 10k | 10 000 | 10 000 | 0 | 0.92 s | **10 886** | 9 ms | 9 ms | 11 ms | 14 ms | 33 ms |
| **READ list** 100 | 100 | 100 | 0 | 29.4 s | **3.4** | 2.89 s | 2.76 s | 3.82 s | 4.54 s | 4.61 s |
| **READ search** 1k | 1 000 | 992 | 8 | 4 m 27 s | **3.74** | 13.25 s | 8.82 s | 27.84 s | 27.97 s | 28.06 s |

Сырые отчёты — `results/baseline_*.json` / `*.log`.

### Что увидел

1. **CREATE** упирается в Go-сервис (multipart парсинг + JSON + sanitizer + два
   `INSERT INTO product` + `INSERT INTO product_status_history`). 8k RPS — терпимо.
   `p99=25ms` ровный.
2. **READ by-id** — primary key lookup. 10k RPS на одном инстансе catalog без
   кэширования — норма.
3. **READ list** — катастрофа. GET `/api/v1/ads` возвращает **все** 100 000
   объявлений одним JSON-массивом (~30 МБ). Не SQL медленный (см. EXPLAIN ниже),
   а:
   - row-scan в pgx → массив `models.Ad`;
   - easyjson-сериализация 30 МБ;
   - передача 30 МБ × 10 параллельных клиентов через loopback.
4. **READ search** — `query=Объявление` матчит все 100k строк, потому что у
   меня все сгенерированные тайтлы начинаются с «Объявление №…». Триграмма
   `<%` отрабатывает, но дальше идёт `GROUP BY p.id` + `COUNT(DISTINCT ...)`
   по 100k групп + 3 LEFT JOIN + ORDER BY. Результат — 13 с avg, 8 запросов
   уложились в 30 c таймаут клиента.

### EXPLAIN — GetAllAds, baseline

```
Sort  (cost=38545.58..38795.58 rows=100000)
  Sort Key: COALESCE(vpp.is_boosted, false) DESC, p.created_at DESC
  Sort Method: external merge  Disk: 31752kB           ← work_mem=4MB не хватает!
  ->  GroupAggregate  (rows=100000)
        Buffers: shared hit=99776
        ->  Merge Left Join (p.id = pi.product_id)
              ->  Index Scan using product_pkey (rows=100000)
                    Filter: deleted_at IS NULL AND status = 'active'
              ->  Index Scan using product_image_product_id_sort_order_key
        + Sort + Seq Scan on favorite (rows=0)         ← у favorite нет индекса по product_id
Execution Time: 298.906 ms
```

Полный файл — `explain/baseline_list.txt`.

### Гипотезы по узким местам

| Проблема | Где видно | Гипотеза фикса |
|----------|-----------|-----------------|
| `external merge Disk: 31MB` | EXPLAIN list | поднять `work_mem`, либо не сортировать 100k — добавить индекс под ORDER BY |
| `Seq Scan on favorite` | EXPLAIN list | индекс `favorite(product_id)` (PK сейчас `(user_id, product_id)`) |
| 30 МБ JSON | wall=2.9 c в loader | LIMIT/OFFSET — главный фикс |
| SearchAds 13 c avg | search log | селективность `<%` низкая на «Объявление», `GROUP BY 100k` непобедим без LIMIT |

---

## Итерация 1 — индексы

### Что сделал

Накатил `perf_test/migrations/iter1_indexes.sql`:

```sql
CREATE INDEX idx_favorite_product_id          ON favorite (product_id);
CREATE INDEX idx_product_active_created_at    ON product (created_at DESC)
    WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX idx_product_seller_id            ON product (seller_id)        WHERE deleted_at IS NULL;
CREATE INDEX idx_product_category_id          ON product (category_id)      WHERE deleted_at IS NULL;
CREATE INDEX idx_product_price_history_product_id  ON product_price_history (product_id, created_at DESC);
CREATE INDEX idx_product_status_history_product_id ON product_status_history (product_id, created_at DESC);
ANALYZE …;
```

Почему именно так:

- `favorite(product_id)` — закрывает Seq Scan в `GetAllAds`/`GetAdByID`.
- `idx_product_active_created_at` — партиальный, под ровно тот WHERE + ORDER BY,
  что в листинге. Шанс дать `Index Scan Backward` без сортировки.
- `seller_id` / `category_id` — для `GetAdsByUserID` и фильтра поиска.
- `*_history(product_id, created_at DESC)` — карточка товара показывает историю
  цены, это запрос по product_id с сортировкой по дате.

### Результаты

| Сценарий | RPS до | RPS после | p99 до | p99 после | Δ |
|----------|--------|-----------|--------|-----------|---|
| CREATE | 8 016 | **7 510** | 25 ms | 28 ms | −6 % (write amplification от +6 индексов) |
| READ by-id | 10 886 | 10 701 | 14 ms | 14 ms | ≈ |
| READ list | 3.40 | 3.33 | 4.54 s | 5.02 s | ≈ |
| READ search | 3.74 | 3.64 | 27.97 s | 28.40 s | ≈ |

Сырые отчёты — `results/iter1_indexes_*`.

### Вывод

Индексы **не дали** ожидаемого выигрыша на листинге. EXPLAIN после `ANALYZE`:

```
Execution Time: 318.119 ms   (было 298.906 ms)
```

Планировщик всё равно идёт по `product_pkey` — потому что `WHERE deleted_at IS NULL
AND status = 'active'` пропускает все 100k строк, а партиальный
`idx_product_active_created_at` для запроса **без LIMIT** хуже PK (надо JOIN-ить
обратно к heap по каждому id). Без LIMIT партиальный индекс бесполезен.

Single-table операции не страдают, но pay-off от индексов виден только когда
запрос реально ограничен по объёму (LIMIT/OFFSET). Переходим к итерации 2.

Полный EXPLAIN — `explain/iter1_list.txt`.

---

## Итерация 2 — пагинация (LIMIT/OFFSET) в листинге

### Что сделал

Код-патч в репозитории и хендлере catalog:

- `services/catalog/internal/repository/ad/postgres.go`: `GetAllAds(ctx, limit, offset)`.
  Запрос теперь — двухступенчатый CTE: сначала `page` вытягивает ровно
  `LIMIT N OFFSET M` id-шников из `product` с тем же ORDER BY, дальше — JOIN
  обратно за полями + аггрегаты только по этим id.
- `services/catalog/internal/usecase/ads/ads.go`: проброс `limit/offset` в storage.
- `services/catalog/internal/delivery/handlers/ads.go`:
  `?limit=N&offset=M` → `parseLimitOffset(r)`. Default 50, max 100 — гвоздями
  прибито в репо (`defaultListLimit`, `maxListLimit`).
- gomock-моки регенерированы (`go generate ./...`), все юнит-тесты зелёные.

Партиальный индекс `idx_product_active_created_at` из итерации 1 теперь
действительно используется планировщиком — `LIMIT 50` останавливает Index Scan
Backward через 50 итераций.

### EXPLAIN — GetAllAds, после LIMIT 50

```
Execution Time: 20.122 ms   ← было 298 ms (без LIMIT)
shared hit=376              ← было 99 776 (-260×)
```

Полный план — `explain/iter2_list.txt`.

### Результаты

Прогнал только read-сценарии — данные те же 100k.

| Сценарий | RPS baseline | RPS iter1 | RPS iter2 | p99 baseline | p99 iter2 | Δ к baseline |
|----------|--------------|-----------|-----------|--------------|-----------|--------------|
| READ by-id | 10 886 | 10 701 | 10 352 | 13.6 ms | 21.0 ms | ≈ |
| **READ list** | **3.40** | 3.33 | **367.53** | **4 539 ms** | **45.6 ms** | **×108 RPS, p99 ÷100** |
| READ search | 3.74 | 3.64 | 3.68 | 27 971 ms | 28 426 ms | ≈ |

Сырьё — `results/iter2_pagination_*`.

### Вывод

LIMIT/OFFSET в листинге — единственный честный фикс. Никакие индексы не спасают
эндпоинт, который дизайнерски возвращает всю таблицу: дальше упирается уже не
в DB, а в JSON-сериализацию + сеть. После пагинации эндпоинт начинает
эксплуатировать партиальный индекс из итерации 1 (без него план снова деградирует
до полного сканирования PK).

Read by-id чуть-чуть просел (p99 14 → 21 мс) из-за нагретого DB и +6 индексов на
write-path, который параллельно дышит. В пределах шума.

---

## Что осталось не оптимизированным и почему

### SearchAds — RPS 3.7, p99 28 s

`GET /api/v1/ads/search?query=Объявление` остался медленным и в baseline, и
после оптимизаций. Причины:

1. **Низкая селективность тестового запроса**. Тайтлы сгенерированы как
   «Объявление №<hex>», слово «Объявление» матчит 100 % таблицы. Триграммный
   `<%` всё равно отрабатывает (GIN), но потом — `GROUP BY p.id` + 3 `LEFT JOIN`
   + `COUNT(DISTINCT)`. Любая агрегация по 100k групп будет десятки секунд.
2. **`SearchAds` не имеет LIMIT во внешнем `SELECT`** (внутренний CTE matched
   ограничен `LIMIT $2`, но дальше JOIN-итерируется по всем матчам). Нужен
   проброс пагинации, как и в `GetAllAds` — следующая итерация.
3. **Запрос написан под селективные поисковые фразы**. На реальных пользователях
   («айфон 13», «мотоблок») селективность ~0.1 %, и GIN отрабатывает за ≤10 мс.
   Тест на «Объявление» — патологический случай, его лечить отдельной формулой:
   обрезать выдачу до 100 после `matched`, не считать `COUNT(DISTINCT pv.id)`
   глобально (или денормализовать в `views_count`).

### Прочее

- `work_mem=4MB` мало, видно по `external merge Disk: 31752kB` в baseline LIST.
  Поднимать в `postgresql.conf` имеет смысл уже после фикса пагинации (без
  LIMIT всё равно сортируется 100k строк → дисковый merge неизбежен).
- `views_count` ведётся колонкой в `product`, но `COUNT(DISTINCT pv.id)` в
  search всё равно идёт. Это техдолг — в search надо взять `p.views_count`.

---

## Сводная таблица

| Итерация | LIST RPS | LIST p99 | CREATE RPS | by-id RPS | search RPS |
|----------|---------:|---------:|-----------:|----------:|-----------:|
| 0 baseline | 3.40 | 4.54 s | 8 016 | 10 886 | 3.74 |
| 1 индексы | 3.33 | 5.02 s | 7 510 | 10 701 | 3.64 |
| **2 пагинация + индексы** | **367.53** | **45 ms** | n/a | 10 352 | 3.68 |

Главный вывод: на read-heavy эндпоинте, который возвращает «всё», порядок
оптимизаций такой —

1. сначала **ограничить выдачу** (LIMIT/OFFSET, cursor pagination);
2. потом подложить под ORDER BY/WHERE **партиальный индекс** под точный фильтр;
3. потом править затратные `COUNT(DISTINCT)`/`GROUP BY` (денормализация).

Индексы без пагинации в этом случае не работают.

## Файлы для повторной защиты

- `init.sql` — DDL до оптимизаций (для проверяющего).
- `migrations/iter1_indexes.sql` — добавленные индексы.
- Код-диф итерации 2 — в коммите ветки `homework/db4` поверх `main`, файлы:
  `services/catalog/internal/repository/ad/postgres.go`,
  `services/catalog/internal/usecase/ads/ads.go`,
  `services/catalog/internal/delivery/handlers/handlers.go`,
  `services/catalog/internal/delivery/handlers/ads.go`,
  +регенерированные моки и тесты.
- `explain/*.txt` — планы PostgreSQL.
- `results/*.json` — машинно-читаемые отчёты loader-а.
