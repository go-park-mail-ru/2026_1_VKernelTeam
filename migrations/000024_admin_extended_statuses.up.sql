-- Расширяем статусы объявления: добавляем admin_deleted, pending_moderation, rejected.
-- Также добавляем поле для аудита удаления админом.

ALTER TABLE product
    DROP CONSTRAINT IF EXISTS product_status_check;

ALTER TABLE product
    ADD CONSTRAINT product_status_check
        CHECK (status IN (
            'draft', 'active', 'reserved', 'sold', 'archived',
            'admin_deleted', 'pending_moderation', 'rejected'
        ));

ALTER TABLE product
    ADD COLUMN deleted_by_admin_id bigint REFERENCES "user" (id),
    ADD COLUMN rejection_reason    text;
