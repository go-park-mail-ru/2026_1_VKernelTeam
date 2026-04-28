-- Auth Service: таблицы пользователей и refresh-токенов
-- Выделено из монолитной миграции 000001_create_initial_tables.up.sql

-- Таблица пользователей
CREATE TABLE "user" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]+$'),
    password_hash text NOT NULL,
    first_name text NOT NULL CHECK (length(first_name) BETWEEN 2 AND 50),
    second_name text CHECK (length(second_name) BETWEEN 2 AND 50),
    avatar_path text,
    rating numeric DEFAULT 0 NOT NULL CHECK (rating >= 0 AND rating <= 5),
    role text NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'support', 'admin')),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Таблица для Refresh-токенов
CREATE TABLE refresh_token (
    token text PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);
