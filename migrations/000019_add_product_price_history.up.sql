CREATE TABLE product_price_history (
    id         bigserial PRIMARY KEY,
    product_id bigint      NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    price      bigint      NOT NULL,
    changed_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_price_history_product_changed
    ON product_price_history (product_id, changed_at);

-- бэкфилл: текущая цена существующих объявлений как первая точка истории
INSERT INTO product_price_history (product_id, price, changed_at)
SELECT id, price, created_at FROM product;

-- триггерная функция: пишем запись при создании объявления
-- и при каждом реальном изменении цены
CREATE OR REPLACE FUNCTION log_product_price_change()
RETURNS trigger AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO product_price_history (product_id, price, changed_at)
        VALUES (NEW.id, NEW.price, NEW.created_at);
    ELSIF (TG_OP = 'UPDATE' AND NEW.price IS DISTINCT FROM OLD.price) THEN
        INSERT INTO product_price_history (product_id, price, changed_at)
        VALUES (NEW.id, NEW.price, NOW());
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_product_price_history
    AFTER INSERT OR UPDATE OF price ON product
    FOR EACH ROW EXECUTE FUNCTION log_product_price_change();
