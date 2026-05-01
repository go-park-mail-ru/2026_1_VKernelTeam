-- 1. Переименование существующих категорий
UPDATE category SET name = 'Запчасти' WHERE name = 'Автозапчасти';
UPDATE category SET name = 'Для дома и дачи' WHERE name = 'Сад и огород';

-- 2. Добавление новых категорий (если отсутствуют)
INSERT INTO category (name)
SELECT 'Авто'
WHERE NOT EXISTS (SELECT 1 FROM category WHERE name = 'Авто');

INSERT INTO category (name)
SELECT 'Работа'
WHERE NOT EXISTS (SELECT 1 FROM category WHERE name = 'Работа');

INSERT INTO category (name)
SELECT 'Товары для детей'
WHERE NOT EXISTS (SELECT 1 FROM category WHERE name = 'Товары для детей');

-- 3. Характеристики для категории "Авто"
DO $$
DECLARE
    cat_id bigint;
BEGIN
    SELECT id INTO cat_id FROM category WHERE name = 'Авто' LIMIT 1;
    IF cat_id IS NOT NULL THEN
        -- Удаляем старые характеристики, если они были добавлены ранее (чтобы избежать дублей)
        DELETE FROM category_characteristic WHERE category_id = cat_id;

        INSERT INTO category_characteristic (category_id, name, allowed_values, sort_order) VALUES
            (cat_id, 'Тип кузова', ARRAY['седан', 'хэтчбек', 'универсал', 'внедорожник', 'кроссовер', 'купе', 'кабриолет', 'микроавтобус', 'пикап', 'другое'], 0),
            (cat_id, 'Марка', NULL, 1),
            (cat_id, 'Модель', NULL, 2),
            (cat_id, 'Год выпуска', NULL, 3),
            (cat_id, 'Пробег (тыс. км)', NULL, 4),
            (cat_id, 'Состояние', ARRAY['новый', 'б/у', 'аварийный', 'требует ремонта'], 5),
            (cat_id, 'Привод', ARRAY['передний', 'задний', 'полный'], 6),
            (cat_id, 'Коробка передач', ARRAY['механика', 'автомат', 'вариатор', 'робот'], 7),
            (cat_id, 'Двигатель', ARRAY['бензин', 'дизель', 'электро', 'гибрид'], 8),
            (cat_id, 'Объём двигателя (л)', NULL, 9),
            (cat_id, 'Цвет', ARRAY['белый', 'чёрный', 'серебристый', 'серый', 'синий', 'красный', 'зелёный', 'жёлтый', 'оранжевый', 'другой'], 10),
            (cat_id, 'Руль', ARRAY['левый', 'правый'], 11),
            (cat_id, 'ПТС', ARRAY['оригинал', 'дубликат', 'электронный'], 12),
            (cat_id, 'Владельцев по ПТС', NULL, 13);
    END IF;
END $$;

-- 4. Характеристики для категории "Работа"
DO $$
DECLARE
    cat_id bigint;
BEGIN
    SELECT id INTO cat_id FROM category WHERE name = 'Работа' LIMIT 1;
    IF cat_id IS NOT NULL THEN
        DELETE FROM category_characteristic WHERE category_id = cat_id;

        INSERT INTO category_characteristic (category_id, name, allowed_values, sort_order) VALUES
            (cat_id, 'Тип занятости', ARRAY['полная', 'частичная', 'удалённая', 'стажировка', 'волонтёрство'], 0),
            (cat_id, 'График работы', ARRAY['5/2', '2/2', '3/3', 'вахта', 'свободный'], 1),
            (cat_id, 'Заработная плата (руб/мес)', NULL, 2),
            (cat_id, 'Опыт работы', ARRAY['без опыта', '1-3 года', '3-6 лет', 'более 6 лет'], 3),
            (cat_id, 'Образование', ARRAY['среднее', 'среднее специальное', 'неполное высшее', 'высшее', 'учёная степень'], 4),
            (cat_id, 'Требования', NULL, 5),
            (cat_id, 'Обязанности', NULL, 6),
            (cat_id, 'Условия работы', NULL, 7);
    END IF;
END $$;

-- 5. Характеристики для категории "Товары для детей"
DO $$
DECLARE
    cat_id bigint;
BEGIN
    SELECT id INTO cat_id FROM category WHERE name = 'Товары для детей' LIMIT 1;
    IF cat_id IS NOT NULL THEN
        DELETE FROM category_characteristic WHERE category_id = cat_id;

        INSERT INTO category_characteristic (category_id, name, allowed_values, sort_order) VALUES
            (cat_id, 'Тип', ARRAY['коляска', 'кроватка', 'автокресло', 'игрушка', 'одежда', 'обувь', 'питание', 'гигиена', 'мебель', 'развивающие', 'ходунки', 'прыгунки', 'манеж', 'горшок', 'велосипед', 'санки', 'другое'], 0),
            (cat_id, 'Возраст', ARRAY['0-3 мес', '3-6 мес', '6-12 мес', '1-2 года', '2-3 года', '3-5 лет', '5-7 лет', '7-12 лет', '12+'], 1),
            (cat_id, 'Состояние', ARRAY['новый', 'б/у отличное', 'б/у хорошее', 'б/у удовлетворительное'], 2),
            (cat_id, 'Бренд', NULL, 3),
            (cat_id, 'Материал', ARRAY['пластик', 'дерево', 'металл', 'текстиль', 'комбинированный', 'другое'], 4),
            (cat_id, 'Цвет', ARRAY['белый', 'чёрный', 'красный', 'синий', 'зелёный', 'жёлтый', 'розовый', 'голубой', 'серый', 'бежевый', 'разноцветный', 'другой'], 5),
            (cat_id, 'Пол ребёнка', ARRAY['мальчик', 'девочка', 'унисекс'], 6),
            (cat_id, 'Безопасность', ARRAY['сертифицирован', 'не сертифицирован'], 7);
    END IF;
END $$;

-- 6. Обновление последовательности (на случай явного указания id)
SELECT setval(pg_get_serial_sequence('category', 'id'), (SELECT coalesce(max(id), 1) FROM category));
