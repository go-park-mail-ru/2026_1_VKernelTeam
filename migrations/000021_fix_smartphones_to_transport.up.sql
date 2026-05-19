-- Миграция 000002 создала «Смартфоны» авто-инкрементом раньше, чем блок
-- INSERT ... OVERRIDING SYSTEM VALUE (3, 'Транспорт'). Конфликт по PK id=3
-- сбросил вставку «Транспорта», а 000009 засеял его характеристики на id=3.

UPDATE product
SET category_id = 1
WHERE category_id = 3
  AND title = 'iPhone 15 Pro';

UPDATE category
SET name = 'Транспорт', parent_id = NULL
WHERE id = 3 AND name = 'Смартфоны';

SELECT setval(pg_get_serial_sequence('category', 'id'), (SELECT coalesce(max(id), 1) FROM category));
