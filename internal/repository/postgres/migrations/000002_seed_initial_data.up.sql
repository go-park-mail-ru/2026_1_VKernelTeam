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
    ('Недвижимость')
ON CONFLICT (name) DO NOTHING;

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

INSERT INTO "user" (id, email, password_hash, first_name) OVERRIDING SYSTEM VALUE VALUES
(101, 'seller101@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller101'),
(102, 'seller102@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller102'),
(103, 'seller103@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller103'),
(104, 'seller104@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller104'),
(105, 'seller105@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller105'),
(106, 'seller106@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller106'),
(107, 'seller107@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller107'),
(108, 'seller108@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller108'),
(109, 'seller109@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller109'),
(110, 'seller110@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller110'),
(111, 'seller111@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller111'),
(112, 'seller112@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller112'),
(113, 'seller113@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller113'),
(114, 'seller114@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller114'),
(115, 'seller115@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller115'),
(116, 'seller116@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller116'),
(117, 'seller117@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller117'),
(118, 'seller118@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller118'),
(119, 'seller119@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller119'),
(120, 'seller120@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqvQdgxbPKq1K1O6nKsH1v1E7qqGC', 'Seller120')
ON CONFLICT DO NOTHING;

INSERT INTO category (id, name) OVERRIDING SYSTEM VALUE VALUES
(1, 'Электроника'),
(2, 'Недвижимость'),
(3, 'Транспорт'),
(4, 'Хобби и отдых'),
(5, 'Музыка'),
(6, 'Ремонт'),
(7, 'Туризм'),
(8, 'Техника для дома'),
(9, 'Игрушки'),
(10, 'Настольные игры')
ON CONFLICT DO NOTHING;

INSERT INTO product (id, seller_id, category_id, title, description, price, status, created_at) OVERRIDING SYSTEM VALUE VALUES
	(1, 101, 1, 'MacBook Pro 16 M1 Max', 'Отличное состояние, полный комплект, использовался только для программирования. Батарея 95%.', 25000000, 'active', CURRENT_TIMESTAMP - INTERVAL '24 hours'),
	(2, 102, 2, 'Диван-кровать Икеа', 'Хорошее состояние, есть небольшие пятна на обивке. Только самовывоз.', 5000000, 'active', CURRENT_TIMESTAMP - INTERVAL '48 hours'),
	(3, 103, 3, 'Горный велосипед Stern', 'Почти новый, катались всего пару раз. Рама 20 дюймов, колеса 27.5.', 5500000, 'active', CURRENT_TIMESTAMP - INTERVAL '72 hours'),
	(4, 104, 4, 'Комплект книг Гарри Поттер', 'Полное собрание от издательства РОСМЭН. Идеальное состояние, как новые.', 0, 'active', CURRENT_TIMESTAMP - INTERVAL '2 hours'),
	(5, 105, 1, 'iPhone 15 Pro Max 256GB', 'Новый, запечатанный. Цвет Natural Titanium. Официальная гарантия.', 12500000, 'active', CURRENT_TIMESTAMP - INTERVAL '1 hours'),
	(6, 106, 2, 'Игровое кресло Cougar', 'Эко-кожа, стальная рама, регулировка 4D подлокотников. Использовалось полгода.', 1850000, 'active', CURRENT_TIMESTAMP - INTERVAL '10 hours'),
	(7, 107, 5, 'Гитара Fender Stratocaster', 'Made in Mexico, 2019 год. Шикарный звук, новые струны Ernie Ball.', 8500000, 'active', CURRENT_TIMESTAMP - INTERVAL '5 hours'),
	(8, 108, 6, 'Набор инструментов 120 предметов', 'Сталь хром-ванадий. Пожизненная гарантия. Почти не пользовались.', 750000, 'active', CURRENT_TIMESTAMP - INTERVAL '15 hours'),
	(9, 109, 1, 'Sony PlayStation 5', 'Версия с дисководом, 2 геймпада DualSense и игра в подарок.', 4900000, 'active', CURRENT_TIMESTAMP - INTERVAL '3 hours'),
	(10, 110, 2, 'Ковер персидский ручной работы', 'Натуральная шерсть. Размер 2х3 метра. Состояние идеальное.', 3000000, 'active', CURRENT_TIMESTAMP - INTERVAL '96 hours'),
	(11, 111, 4, 'Коллекционное издание Булгакова', 'Кожаный переплет, золотое тиснение. Ограниченная серия.', 1500000, 'active', CURRENT_TIMESTAMP - INTERVAL '20 hours'),
	(12, 112, 7, 'Палатка 3-х местная Tramp', 'Двухслойная, влагозащита 5000мм. Идеальна для походов в горы.', 1200000, 'active', CURRENT_TIMESTAMP - INTERVAL '12 hours'),
	(13, 113, 1, 'Монитор LG UltraWide 34"', 'IPS матрица, 144Hz. Широкий формат для работы и игр.', 4500000, 'active', CURRENT_TIMESTAMP - INTERVAL '18 hours'),
	(14, 114, 3, 'Сноуборд Burton Custom', 'Универсальный борд, прогиб Camber. Крепления в комплекте.', 3500000, 'active', CURRENT_TIMESTAMP - INTERVAL '8 hours'),
	(15, 115, 8, 'Кофемашина DeLonghi Magnifica', 'Автоматический капучинатор. Кофе как в кофейне.', 2800000, 'active', CURRENT_TIMESTAMP - INTERVAL '4 hours'),
	(16, 116, 1, 'AirPods Pro 2 (USB-C)', 'Оригинал, новые в пленке. Активное шумоподавление и прозрачность.', 2100000, 'active', CURRENT_TIMESTAMP - INTERVAL '0.5 hours'),
	(17, 117, 5, 'Синтезатор Yamaha PSR', '61 клавиша, чувствительность к нажатию. Идеально для обучения.', 3200000, 'active', CURRENT_TIMESTAMP - INTERVAL '60 hours'),
	(18, 118, 9, 'Lego Technic Самосвал', 'Огромная модель, все механизмы работают. Инструкция в наличии.', 1400000, 'active', CURRENT_TIMESTAMP - INTERVAL '22 hours'),
	(19, 119, 1, 'Механическая клавиатура Keychron', 'Алюминиевый корпус, Hot-swap, RGB подсветка. Смазанные переключатели.', 1600000, 'active', CURRENT_TIMESTAMP - INTERVAL '6 hours'),
	(20, 120, 10, 'Настольная игра ''Gloomhaven''', 'Самая масштабная настолка. Огромная коробка весом 10кг. Состояние 5/5.', 1450000, 'active', CURRENT_TIMESTAMP - INTERVAL '14 hours')
ON CONFLICT DO NOTHING;

INSERT INTO product_image (product_id, file_path, sort_order) VALUES
	(1, '/static/img/1.webp', 0),
	(2, '/static/img/2.webp', 0),
	(3, '/static/img/3.webp', 0),
	(4, '/static/img/4.webp', 0),
	(5, '/static/img/5.webp', 0),
	(5, '/static/img/6.webp', 1),
	(6, '/static/img/7.webp', 0),
	(7, '/static/img/8.webp', 0),
	(7, '/static/img/9.webp', 1),
	(8, '/static/img/10.webp', 0),
	(9, '/static/img/11.webp', 0),
	(10, '/static/img/12.webp', 0),
	(11, '/static/img/13.webp', 0),
	(12, '/static/img/14.webp', 0),
	(13, '/static/img/15.webp', 0),
	(14, '/static/img/16.webp', 0),
	(15, '/static/img/17.webp', 0),
	(16, '/static/img/18.webp', 0),
	(17, '/static/img/19.webp', 0),
	(18, '/static/img/20.webp', 0),
	(19, '/static/img/21.webp', 0),
	(20, '/static/img/22.webp', 0)
ON CONFLICT DO NOTHING;

SELECT setval(pg_get_serial_sequence('"user"', 'id'), (SELECT coalesce(max(id), 1) FROM "user"));
SELECT setval(pg_get_serial_sequence('category', 'id'), (SELECT coalesce(max(id), 1) FROM category));
SELECT setval(pg_get_serial_sequence('product', 'id'), (SELECT coalesce(max(id), 1) FROM product));
