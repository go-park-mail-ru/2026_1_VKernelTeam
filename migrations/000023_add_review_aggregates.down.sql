DROP TRIGGER IF EXISTS review_aggregates_trigger ON review;
DROP FUNCTION IF EXISTS trg_review_aggregates();
DROP FUNCTION IF EXISTS recalc_user_review_aggregates(bigint);
DROP INDEX IF EXISTS idx_review_receiver_id_id;
DROP INDEX IF EXISTS idx_review_sender_id_id;
ALTER TABLE "user" DROP COLUMN IF EXISTS reviews_count;
