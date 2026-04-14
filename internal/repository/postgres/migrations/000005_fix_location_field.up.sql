-- 1. Добавляем колонку, если вдруг её нет (безопасный способ)
ALTER TABLE product ADD COLUMN IF NOT EXISTS location text;

-- 2. Заполняем пустые значения дефолтным городом,
-- чтобы Go не падал при сканировании в обычную строку (string)
UPDATE product SET location = 'Москва' WHERE location IS NULL;
