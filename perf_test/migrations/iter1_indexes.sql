-- Итерация 1: индексы для оптимизации чтения объявлений.
-- Применяется отдельно от основных миграций сервиса, чтобы можно было
-- легко включать/откатывать в рамках нагрузочных тестов.

-- favorite.product_id: PK = (user_id, product_id), обратный поиск по product_id
-- упирается в Seq Scan на каждом JOIN из product.
CREATE INDEX IF NOT EXISTS idx_favorite_product_id
  ON favorite (product_id);

-- Партиальный индекс под основной WHERE-фильтр листинга:
-- deleted_at IS NULL AND status = 'active' покрывает ~100% строк в проде,
-- но даёт ORDER BY-friendly путь без сортировки на диске.
CREATE INDEX IF NOT EXISTS idx_product_active_created_at
  ON product (created_at DESC)
  WHERE deleted_at IS NULL AND status = 'active';

-- Объявления конкретного продавца / поиск по seller_id (GetAdsByUserID).
CREATE INDEX IF NOT EXISTS idx_product_seller_id
  ON product (seller_id)
  WHERE deleted_at IS NULL;

-- Категория — частый фильтр в SearchAds (CategoryID).
CREATE INDEX IF NOT EXISTS idx_product_category_id
  ON product (category_id)
  WHERE deleted_at IS NULL;

-- product_status_history / product_price_history часто JOIN-ятся по product_id
-- (история цены показывается на странице карточки).
CREATE INDEX IF NOT EXISTS idx_product_price_history_product_id
  ON product_price_history (product_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_product_status_history_product_id
  ON product_status_history (product_id, created_at DESC);

ANALYZE product, product_image, favorite, product_price_history, product_status_history;
