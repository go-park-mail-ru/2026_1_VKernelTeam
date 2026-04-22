ALTER TABLE product ADD COLUMN views_count bigint NOT NULL DEFAULT 0;

UPDATE product p
SET views_count = (
    SELECT COUNT(*)
    FROM product_view pv
    WHERE pv.product_id = p.id
);

CREATE INDEX idx_product_view_product_viewed
    ON product_view (product_id, viewed_at);
