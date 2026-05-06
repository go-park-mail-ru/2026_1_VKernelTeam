DROP TRIGGER IF EXISTS cart_count_trigger ON cart_item;
DROP FUNCTION IF EXISTS update_cart_count();

ALTER TABLE "user" DROP COLUMN IF EXISTS cart_count;
