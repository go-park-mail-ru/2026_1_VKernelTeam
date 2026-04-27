CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_product_title_trgm ON product USING GIN (title gin_trgm_ops);
CREATE INDEX idx_product_description_trgm ON product USING GIN (description gin_trgm_ops);
