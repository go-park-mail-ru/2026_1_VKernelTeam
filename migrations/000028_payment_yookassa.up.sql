-- Добавляем поля под асинхронного провайдера (ЮКасса):
--   confirmation_url     — URL, на который фронт редиректит пользователя;
--   raw_provider_payload — последний полученный payload провайдера (для аудита и расследований).
ALTER TABLE payment
    ADD COLUMN IF NOT EXISTS confirmation_url     text,
    ADD COLUMN IF NOT EXISTS raw_provider_payload jsonb;

-- Уникальность пары (provider, provider_ref) для идемпотентного апдейта по webhook.
-- Partial-индекс: NULL provider_ref у только что созданных pending-платежей не блокируется.
CREATE UNIQUE INDEX IF NOT EXISTS payment_provider_ref_uniq
    ON payment (provider, provider_ref)
    WHERE provider_ref IS NOT NULL;

-- Индекс для cron-сверщика зависших платежей.
CREATE INDEX IF NOT EXISTS idx_payment_status_provider_created
    ON payment (status, provider, created_at)
    WHERE status = 'pending';
