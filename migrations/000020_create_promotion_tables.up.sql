-- Платное продвижение объявлений: тарифы, купленные промо, кошелёк, лог транзакций, лог платежей.
-- Все суммы — целые рубли (bigint, без копеек).

-- 1. Каталог тарифов продвижения.
CREATE TABLE promotion_plan (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code          text NOT NULL UNIQUE CHECK (length(code) BETWEEN 1 AND 64),
    kind          text NOT NULL CHECK (kind IN ('boost', 'highlight')),
    duration_days int  NOT NULL CHECK (duration_days > 0),
    price         bigint NOT NULL CHECK (price >= 0),
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Купленные промо. Одно объявление может иметь активные boost и highlight параллельно.
CREATE TABLE promotion (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES product (id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT,
    plan_id    bigint NOT NULL REFERENCES promotion_plan (id) ON DELETE RESTRICT,
    kind       text   NOT NULL CHECK (kind IN ('boost', 'highlight')),
    starts_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz NOT NULL,
    price_paid bigint NOT NULL CHECK (price_paid >= 0),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Композитный индекс под основной запрос: «активные промо для объявления и типа».
CREATE INDEX idx_promotion_product_kind_expires
    ON promotion (product_id, kind, expires_at DESC);

-- Дополнительный индекс под историю пользователя.
CREATE INDEX idx_promotion_user_created
    ON promotion (user_id, created_at DESC);

-- 3. Кошелёк пользователя. Создаётся лениво при первом списании/пополнении.
CREATE TABLE wallet (
    user_id    bigint PRIMARY KEY REFERENCES "user" (id) ON DELETE CASCADE,
    balance    bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 4. Лог операций по кошельку (append-only, идемпотентный).
CREATE TABLE wallet_transaction (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         bigint NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT,
    amount          bigint NOT NULL,
    type            text   NOT NULL CHECK (type IN ('topup', 'promotion_charge', 'refund')),
    reference_id    bigint,
    idempotency_key text UNIQUE,
    created_at      timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wallet_transaction_user_created
    ON wallet_transaction (user_id, created_at DESC);

-- 5. Платежи (лог пополнений через провайдера).
CREATE TABLE payment (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      bigint NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT,
    amount       bigint NOT NULL CHECK (amount > 0),
    status       text   NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed', 'cancelled')),
    provider     text   NOT NULL DEFAULT 'mock',
    provider_ref text,
    created_at   timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- VIEW для каталога: плоские флаги активных промо по объявлению.
CREATE VIEW v_product_promotion AS
SELECT
    product_id,
    bool_or(kind = 'boost'     AND expires_at > CURRENT_TIMESTAMP) AS is_boosted,
    bool_or(kind = 'highlight' AND expires_at > CURRENT_TIMESTAMP) AS is_highlighted,
    MAX(CASE WHEN kind = 'boost'     AND expires_at > CURRENT_TIMESTAMP THEN expires_at END) AS boost_expires_at,
    MAX(CASE WHEN kind = 'highlight' AND expires_at > CURRENT_TIMESTAMP THEN expires_at END) AS highlight_expires_at
FROM promotion
GROUP BY product_id;

-- Стартовые тарифы. Цены в рублях, целые.
INSERT INTO promotion_plan (code, kind, duration_days, price) VALUES
    ('boost_1d',     'boost',     1,  49),
    ('boost_7d',     'boost',     7,  199),
    ('highlight_1d', 'highlight', 1,  29),
    ('highlight_7d', 'highlight', 7,  99);
