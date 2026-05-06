-- 1. Удаляем "пустое объявление" с ценой 1000 копеек (10 рублей), если оно существует
DELETE FROM product WHERE price = 1000;

-- 2. Конвертируем все цены из копеек в рубли (деление на 100)
UPDATE product SET price = price / 100;

-- 3. Заполняем location разными городами
UPDATE product SET location =
  CASE mod(id, 10)
    WHEN 0 THEN 'Москва'
    WHEN 1 THEN 'Санкт-Петербург'
    WHEN 2 THEN 'Казань'
    WHEN 3 THEN 'Новосибирск'
    WHEN 4 THEN 'Екатеринбург'
    WHEN 5 THEN 'Нижний Новгород'
    WHEN 6 THEN 'Сочи'
    WHEN 7 THEN 'Владивосток'
    WHEN 8 THEN 'Калининград'
    WHEN 9 THEN 'Ростов-на-Дону'
  END;

UPDATE product SET updated_at = CURRENT_TIMESTAMP;
