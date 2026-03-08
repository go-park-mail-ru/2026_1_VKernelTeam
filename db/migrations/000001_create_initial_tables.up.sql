-- Таблица пользователей
CREATE TABLE "user" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    first_name text NOT NULL,
    second_name text,
    avatar_url text,
    rating numeric DEFAULT 0 NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица категорий (рекурсивная связь для подкатегорий)
CREATE TABLE category (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL UNIQUE,
    parent_id bigint REFERENCES category (id) ON DELETE SET NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица объявлений/товаров
CREATE TABLE product (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seller_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    category_id bigint NOT NULL REFERENCES category (id) ON DELETE RESTRICT,
    title text NOT NULL,
    description text NOT NULL,
    price bigint NOT NULL,
    status text NOT NULL,
    views_count bigint DEFAULT 0 NOT NULL,
    favorites_count bigint DEFAULT 0 NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamptz
);

-- Изображения товаров
CREATE TABLE product_image (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    url text NOT NULL,
    is_main boolean DEFAULT FALSE NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Избранное (составной PK)
CREATE TABLE favorite (
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, product_id)
);

-- Корзина (составной PK)
CREATE TABLE cart_item (
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
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
    rating integer NOT NULL CHECK (rating >= 1 AND rating <= 5),
    content text NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Чаты
CREATE TABLE chat (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    buyer_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    seller_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Сообщения
CREATE TABLE message (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    chat_id bigint NOT NULL REFERENCES chat (id) ON DELETE CASCADE,
    sender_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    text_content text NOT NULL,
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
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- История статуса товара
CREATE TABLE product_status_history (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    old_status text NOT NULL,
    new_status text NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);
