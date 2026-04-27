DROP INDEX IF EXISTS idx_product_view_product_viewed;
ALTER TABLE product DROP COLUMN IF EXISTS views_count;
