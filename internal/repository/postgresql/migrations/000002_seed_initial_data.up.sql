-- Создание начальных пользователей
INSERT INTO "user" (email, password_hash, first_name, second_name)
VALUES
    ('admin@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Иван', 'Иванов'),
    ('buyer@example.com', '$2a$10$K4GfjvPtKhVqYdE.YmVdB.7xOdIgK1KxHpVqkXs8vQ8xPqVKjKjKy', 'Петр', 'Петров')
ON CONFLICT (email) DO NOTHING;

-- Создание категорий
INSERT INTO category (name)
VALUES
    ('Электроника'),
    ('Недвижимость');

INSERT INTO category (name, parent_id)
VALUES (
    'Смартфоны',
    (SELECT id FROM category WHERE name = 'Электроника')
)
ON CONFLICT (name) DO NOTHING;
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
