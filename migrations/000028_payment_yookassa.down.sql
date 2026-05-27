DROP INDEX IF EXISTS idx_payment_status_provider_created;
DROP INDEX IF EXISTS payment_provider_ref_uniq;

ALTER TABLE payment
    DROP COLUMN IF EXISTS raw_provider_payload,
    DROP COLUMN IF EXISTS confirmation_url;
