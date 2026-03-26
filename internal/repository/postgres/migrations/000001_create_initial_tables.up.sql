-- Таблица пользователей с ограничениями длины и валидацией email
CREATE TABLE "user" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]+$'),
    password_hash text NOT NULL,
    first_name text NOT NULL CHECK (length(first_name) BETWEEN 2 AND 50),
    second_name text CHECK (length(second_name) BETWEEN 2 AND 50),
    avatar_path text,
    rating numeric DEFAULT 0 NOT NULL CHECK (rating >= 0 AND rating <= 5),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица для Refresh-токенов (для безопасной авторизации по ТЗ)
CREATE TABLE refresh_token (
    token text PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица категорий (иерархическая)
CREATE TABLE category (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL UNIQUE CHECK (length(name) BETWEEN 2 AND 100),
    parent_id bigint REFERENCES category (id) ON DELETE SET NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица товаров
CREATE TABLE product (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seller_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    category_id bigint NOT NULL REFERENCES category (id) ON DELETE RESTRICT,
    title text NOT NULL CHECK (length(title) BETWEEN 5 AND 150),
    description text NOT NULL CHECK (length(description) BETWEEN 10 AND 5000),
    price bigint NOT NULL CHECK (price >= 0),
    status text NOT NULL CHECK (status IN ('draft', 'active', 'reserved', 'sold', 'archived')),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamptz
);

-- Лог просмотров (Решение проблемы MVCC)
CREATE TABLE product_view (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    user_id bigint REFERENCES "user" (id) ON DELETE SET NULL,
    viewed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Изображения товаров
CREATE TABLE product_image (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    file_path text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (product_id, sort_order)
);

-- Заказы (Формирование сделки)
CREATE TABLE "order" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    buyer_id bigint NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT,
    total_amount bigint NOT NULL CHECK (total_amount >= 0),
    status text NOT NULL CHECK (status IN ('created', 'paid', 'delivering', 'completed', 'cancelled')),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE order_item (
    order_id bigint NOT NULL REFERENCES "order" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE RESTRICT,
    price_at_purchase bigint NOT NULL CHECK (price_at_purchase >= 0),
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    PRIMARY KEY (order_id, product_id)
);

-- Избранное
CREATE TABLE favorite (
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, product_id)
);

-- Корзина
CREATE TABLE cart_item (
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    quantity integer DEFAULT 1 NOT NULL CHECK (quantity > 0),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, product_id)
);

-- Отзывы
CREATE TABLE review (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sender_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    receiver_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    rating integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    content text NOT NULL CHECK (length(content) BETWEEN 5 AND 2000),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (sender_id, product_id, receiver_id),
    CHECK (sender_id != receiver_id)
);

-- Чаты
CREATE TABLE chat (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    buyer_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    seller_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (product_id, buyer_id, seller_id),
    CHECK (buyer_id != seller_id)
);

-- Сообщения
CREATE TABLE message (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    chat_id bigint NOT NULL REFERENCES chat (id) ON DELETE CASCADE,
    sender_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    text_content text NOT NULL CHECK (length(text_content) > 0),
    is_read boolean DEFAULT FALSE NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- История цены товара
CREATE TABLE product_price_history (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    old_price bigint NOT NULL,
    new_price bigint NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- История статуса товара
CREATE TABLE product_status_history (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    old_status text NOT NULL,
    new_status text NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);
