DROP INDEX IF EXISTS idx_order_buyer_id_desc;
ALTER TABLE "order" DROP CONSTRAINT IF EXISTS order_source_check;
ALTER TABLE "order" DROP COLUMN IF EXISTS chat_id;
ALTER TABLE "order" DROP COLUMN IF EXISTS source;
