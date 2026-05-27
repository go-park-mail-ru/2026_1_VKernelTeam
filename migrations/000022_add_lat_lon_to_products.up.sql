-- Координаты объявления (выбираются через Яндекс Карты на фронте).
-- NULL допустимы: старые объявления без координат остаются валидными.
ALTER TABLE product
    ADD COLUMN lat double precision,
    ADD COLUMN lon double precision;

ALTER TABLE product
    ADD CONSTRAINT product_lat_range CHECK (lat IS NULL OR (lat BETWEEN -90 AND 90)),
    ADD CONSTRAINT product_lon_range CHECK (lon IS NULL OR (lon BETWEEN -180 AND 180));
