-- Миграция: Замена префикса путей к изображениям для товаров 1-20
-- Изменяет /static/img/ на https://clover-media-storage.hb.vkcs.cloud/ads/

UPDATE product_image
SET file_path = REPLACE(file_path, '/static/img/', 'https://clover-media-storage.hb.vkcs.cloud/ads/')
WHERE product_id BETWEEN 1 AND 20
  AND file_path LIKE '/static/img/%';
