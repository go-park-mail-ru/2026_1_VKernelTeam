-- =====================================================================
--  Гранты на конкретные таблицы для сервисных ролей.
-- ---------------------------------------------------------------------
--  Запускается ПОСЛЕ миграций (golang-migrate) — иначе таблиц
--  ещё не существует и GRANT'ы упадут. Запуск выполняется отдельным
--  контейнером `db_grants` в docker-compose.yaml (depends_on:
--  db_migrate.service_completed_successfully).
--
--  Идемпотентность: GRANT в postgres сам по себе идемпотентен —
--  повторная выдача того же гранта ничего не ломает. Поэтому скрипт
--  можно безопасно перезапускать сколько угодно раз.
-- =====================================================================

\set ON_ERROR_STOP on

-- ---------------------------------------------------------------------
-- AUTH SERVICE
-- ---------------------------------------------------------------------
-- Работа с таблицей "user" и refresh-токенами.
-- DELETE на "user" не выдаём: удаление реализовано как soft-delete.
GRANT SELECT, INSERT, UPDATE ON TABLE "user" TO auth_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE refresh_token TO auth_app;

-- ---------------------------------------------------------------------
-- CATALOG SERVICE
-- ---------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
      product,
      product_image,
      product_view,
      product_price_history,
      product_status_history,
      product_characteristic,
      product_custom_characteristic,
      favorite,
      review
TO catalog_app;

-- Справочники — только чтение, меняются миграциями.
GRANT SELECT ON TABLE
      category,
      category_characteristic
TO catalog_app;

-- Чтение профилей продавцов для отображения карточек объявлений.
GRANT SELECT ON TABLE "user" TO catalog_app;

-- ---------------------------------------------------------------------
-- COMMERCE SERVICE
-- ---------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
      cart_item,
      "order",
      order_item,
      chat,
      message
TO commerce_app;

-- Чтение карточек товаров для оформления заказа.
GRANT SELECT ON TABLE
      product,
      product_image,
      "user"
TO commerce_app;

-- Снятие/постановка резерва (status active -> reserved -> sold).
GRANT UPDATE (status, updated_at) ON TABLE product TO commerce_app;

-- ---------------------------------------------------------------------
-- SUPPORT SERVICE
-- ---------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
      support_ticket,
      support_message
TO support_app;

-- Чтение профилей для отображения авторов тикетов.
GRANT SELECT ON TABLE "user" TO support_app;

-- ---------------------------------------------------------------------
-- pg_stat_statements: только мониторинг
-- ---------------------------------------------------------------------
-- Расширение создано миграцией 000019. Запрещаем чтение всем подряд
-- (чужие SQL-запросы могут косвенно раскрыть структуру и фильтры),
-- мониторингу — разрешаем явно.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'pg_stat_statements') THEN
        EXECUTE 'REVOKE ALL ON TABLE pg_stat_statements      FROM PUBLIC';
        EXECUTE 'REVOKE ALL ON TABLE pg_stat_statements_info FROM PUBLIC';
        EXECUTE 'GRANT  SELECT ON pg_stat_statements         TO clover_monitoring';
        EXECUTE 'GRANT  SELECT ON pg_stat_statements_info    TO clover_monitoring';
    END IF;
END
$$;
