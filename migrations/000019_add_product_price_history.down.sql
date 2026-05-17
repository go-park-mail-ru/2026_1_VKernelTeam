DROP TRIGGER IF EXISTS trg_product_price_history ON product;
DROP FUNCTION IF EXISTS log_product_price_change();
DROP TABLE IF EXISTS product_price_history;
