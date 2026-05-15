ALTER TABLE "user" ADD COLUMN IF NOT EXISTS cart_count integer DEFAULT 0 NOT NULL;

CREATE OR REPLACE FUNCTION update_cart_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE "user" SET cart_count = cart_count + 1 WHERE id = NEW.user_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE "user" SET cart_count = cart_count - 1 WHERE id = OLD.user_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cart_count_trigger
AFTER INSERT OR DELETE ON cart_item
FOR EACH ROW
EXECUTE FUNCTION update_cart_count();
