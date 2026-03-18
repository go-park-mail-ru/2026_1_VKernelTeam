-- Создание начальных пользователей
INSERT INTO "user" (email, password_hash, first_name, second_name)
VALUES
    ('admin@example.com', 'hash_admin_123', 'Иван', 'Иванов'),
    ('buyer@example.com', 'hash_buyer_456', 'Петр', 'Петров');

-- Создание категорий
INSERT INTO category (name)
VALUES
    ('Электроника'),
    ('Недвижимость');

-- Создание подкатегории
INSERT INTO category (name, parent_id)
SELECT 'Смартфоны', id FROM category WHERE name = 'Электроника';

-- Добавление товара
INSERT INTO product (seller_id, category_id, title, description, price, status)
VALUES (
    (SELECT id FROM "user" WHERE email = 'admin@example.com'),
    (SELECT id FROM category WHERE name = 'Смартфоны'),
    'iPhone 15 Pro',
    'В отличном состоянии, полный комплект',
    9500000, -- цена в копейках
    'active'
);
