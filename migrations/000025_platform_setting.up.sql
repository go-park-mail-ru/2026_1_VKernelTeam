-- Глобальные настройки платформы (ключ-значение).
-- Используется для флага модерации и идентификатора системного пользователя.

CREATE TABLE platform_setting (
    key        text PRIMARY KEY,
    value      text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    updated_by bigint REFERENCES "user" (id)
);

INSERT INTO platform_setting (key, value)
VALUES ('moderation_enabled', 'false')
ON CONFLICT (key) DO NOTHING;
