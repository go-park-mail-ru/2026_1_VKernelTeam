ALTER TABLE product
    DROP COLUMN IF EXISTS deleted_by_admin_id,
    DROP COLUMN IF EXISTS rejection_reason;

ALTER TABLE product
    DROP CONSTRAINT IF EXISTS product_status_check;

ALTER TABLE product
    ADD CONSTRAINT product_status_check
        CHECK (status IN ('draft', 'active', 'reserved', 'sold', 'archived'));
