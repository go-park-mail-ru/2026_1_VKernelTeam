-- =====================================================================
--  Создание сервисных пользователей и распределение прав.
-- =====================================================================
--  Назначение:
--    Каждый микросервис подключается к общей PostgreSQL под собственной
--    учётной записью с минимальным набором прав (Principle of Least
--    Privilege). Это снижает blast radius: компрометация одного сервиса
--    не даёт атакующему доступа ко всей БД.
--
--  Архитектура ролей:
--    * Базовая роль `app_base` — общие права на схему public + базовый
--      набор справочников (user, product, category). Сервисные роли её
--      наследуют, чтобы не дублировать одни и те же гранты.
--    * Сервисные роли `auth_app`, `catalog_app`, `commerce_app`,
--      `support_app` — конкретные права на таблицы каждого сервиса.
--    * Отдельная роль `clover_monitoring` — для Prometheus
--      postgres_exporter / DBA: только чтение pg_stat_*, без доступа
--      к пользовательским данным.
--
--  Запуск:
--    Скрипт идемпотентен — повторный запуск не ломает существующих
--    пользователей (используется DO-блок с проверкой через pg_roles).
--    Пароли передаются через переменные psql, например:
--      psql -v auth_pass="$AUTH_DB_PASSWORD" \
--           -v catalog_pass="$CATALOG_DB_PASSWORD" \
--           -v commerce_pass="$COMMERCE_DB_PASSWORD" \
--           -v support_pass="$SUPPORT_DB_PASSWORD" \
--           -v monitoring_pass="$MONITORING_DB_PASSWORD" \
--           -f 01_create_app_users.sql
--    В docker-compose скрипт подключается через volume в /docker-
--    entrypoint-initdb.d/ — postgres-image выполняет его один раз при
--    инициализации тома.
-- =====================================================================

-- ---------------------------------------------------------------------
-- 0. Подготовка: используем безопасные настройки внутри скрипта
-- ---------------------------------------------------------------------
\set ON_ERROR_STOP on

-- ---------------------------------------------------------------------
-- 1. Базовая роль (NOLOGIN — это группа, не пользователь)
-- ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_base') THEN
        CREATE ROLE app_base NOLOGIN;
    END IF;
END
$$;

-- Право подключаться к БД получают только наследники app_base.
REVOKE ALL ON DATABASE clover FROM PUBLIC;
GRANT  CONNECT ON DATABASE clover TO app_base;

-- В схеме public сидят все таблицы приложения.
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT  USAGE ON SCHEMA public TO app_base;

-- ---------------------------------------------------------------------
-- 2. Сервисные роли (LOGIN, с паролями из переменных)
-- ---------------------------------------------------------------------
--   auth_app    — работает с таблицей "user" (регистрация, профиль).
--   catalog_app — товары, фото, избранное, характеристики, просмотры.
--   commerce_app — корзина, заказы, чаты, сообщения.
--   support_app — обращения в поддержку.
--
-- psql-подстановка `:'var'` НЕ работает внутри dollar-quoted блоков
-- ($$ ... $$), поэтому пароли заливаем сначала в custom GUC через
-- SET, а потом читаем из DO-блока через current_setting(). Префикс
-- 'app.' допустим без объявления в custom_variable_classes (PG 10+).
-- ---------------------------------------------------------------------

SET app.auth_pass        = :'auth_pass';
SET app.catalog_pass     = :'catalog_pass';
SET app.commerce_pass    = :'commerce_pass';
SET app.support_pass     = :'support_pass';
SET app.monitoring_pass  = :'monitoring_pass';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_app') THEN
        EXECUTE format(
            'CREATE ROLE auth_app LOGIN PASSWORD %L IN ROLE app_base',
            current_setting('app.auth_pass')
        );
    ELSE
        EXECUTE format(
            'ALTER ROLE auth_app WITH PASSWORD %L',
            current_setting('app.auth_pass')
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'catalog_app') THEN
        EXECUTE format(
            'CREATE ROLE catalog_app LOGIN PASSWORD %L IN ROLE app_base',
            current_setting('app.catalog_pass')
        );
    ELSE
        EXECUTE format(
            'ALTER ROLE catalog_app WITH PASSWORD %L',
            current_setting('app.catalog_pass')
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'commerce_app') THEN
        EXECUTE format(
            'CREATE ROLE commerce_app LOGIN PASSWORD %L IN ROLE app_base',
            current_setting('app.commerce_pass')
        );
    ELSE
        EXECUTE format(
            'ALTER ROLE commerce_app WITH PASSWORD %L',
            current_setting('app.commerce_pass')
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'support_app') THEN
        EXECUTE format(
            'CREATE ROLE support_app LOGIN PASSWORD %L IN ROLE app_base',
            current_setting('app.support_pass')
        );
    ELSE
        EXECUTE format(
            'ALTER ROLE support_app WITH PASSWORD %L',
            current_setting('app.support_pass')
        );
    END IF;
END
$$;

-- Жёстко ограничиваем число параллельных соединений на сервисную роль.
-- Это последняя линия защиты от утечки соединений в приложении.
-- Подробнее про расчёт — в db/README.md (раздел connection pool).
ALTER ROLE auth_app     CONNECTION LIMIT 25;
ALTER ROLE catalog_app  CONNECTION LIMIT 35;
ALTER ROLE commerce_app CONNECTION LIMIT 35;
ALTER ROLE support_app  CONNECTION LIMIT 15;

-- ---------------------------------------------------------------------
-- 3. Гранты на таблицы
-- ---------------------------------------------------------------------
-- Сами гранты на конкретные таблицы вынесены в отдельный скрипт
-- db/security/02_grant_table_privileges.sql, который запускается
-- ПОСЛЕ миграций (контейнер `db_grants` в docker-compose). Причина:
-- этот файл выполняется postgres-image на этапе initdb, до того как
-- какие-либо таблицы в БД появятся. Поэтому здесь выдаются только
-- права на БД и схему, а на таблицы — отдельно после миграций.

-- ---------------------------------------------------------------------
-- 4. Мониторинговая роль
-- ---------------------------------------------------------------------
-- Используется postgres_exporter и pgbadger.
-- pg_monitor — встроенная роль (PG 10+), даёт доступ к pg_stat_*,
-- pg_stat_statements и системным каталогам без прав на сами данные.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clover_monitoring') THEN
        EXECUTE format(
            'CREATE ROLE clover_monitoring LOGIN PASSWORD %L',
            current_setting('app.monitoring_pass')
        );
    ELSE
        EXECUTE format(
            'ALTER ROLE clover_monitoring WITH PASSWORD %L',
            current_setting('app.monitoring_pass')
        );
    END IF;
END
$$;

GRANT CONNECT ON DATABASE clover TO clover_monitoring;
GRANT USAGE   ON SCHEMA   public TO clover_monitoring;
GRANT pg_monitor                 TO clover_monitoring;

-- pg_stat_statements view создаётся миграцией (000019), которая идёт
-- после init-фазы. Гранты на эту view выдаются в 02_grant_table_privileges.sql.

-- ---------------------------------------------------------------------
-- 10. Default privileges на будущие таблицы
-- ---------------------------------------------------------------------
-- Когда новая миграция создаёт таблицу под суперпользователем postgres,
-- сервисные роли по умолчанию не получают на неё прав. Это намеренно:
-- права выдаются явно в самой миграции (см. db/migrations/*),
-- чтобы ревьюер видел изменение в diff.
--
-- Если хочется автоматизировать SELECT для catalog/commerce/support —
-- можно раскомментировать блок ниже, но не делаем этого по соображениям
-- security-by-explicit:
--
--   ALTER DEFAULT PRIVILEGES IN SCHEMA public
--       GRANT SELECT ON TABLES TO clover_monitoring;
