ALTER TABLE product
    DROP CONSTRAINT IF EXISTS product_lat_range,
    DROP CONSTRAINT IF EXISTS product_lon_range;

ALTER TABLE product
    DROP COLUMN IF EXISTS lat,
    DROP COLUMN IF EXISTS lon;
