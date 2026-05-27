-- Создаём системного пользователя для отправки сообщений от имени администрации.
-- ID автогенерируется (IDENTITY ALWAYS), запоминаем его в platform_setting.

WITH new_user AS (
    INSERT INTO "user" (email, password_hash, first_name, second_name, role)
    VALUES ('system@clover.local', 'NO_LOGIN', 'Администрация', 'Clover', 'admin')
    ON CONFLICT (email) DO UPDATE SET role = EXCLUDED.role
    RETURNING id
)
INSERT INTO platform_setting (key, value)
SELECT 'system_user_id', new_user.id::text FROM new_user
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
