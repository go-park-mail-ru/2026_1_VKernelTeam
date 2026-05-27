-- Поле для агрегата количества отзывов о пользователе.
ALTER TABLE "user" ADD COLUMN reviews_count integer NOT NULL DEFAULT 0;

-- Бэкфилл для существующих данных.
UPDATE "user" u
SET reviews_count = sub.cnt,
    rating = COALESCE(sub.avg_rating, 0)
FROM (
    SELECT receiver_id, COUNT(*) AS cnt, AVG(rating)::numeric(3,2) AS avg_rating
    FROM review
    GROUP BY receiver_id
) sub
WHERE u.id = sub.receiver_id;

-- Индексы для пагинации и обращений.
CREATE INDEX IF NOT EXISTS idx_review_receiver_id_id ON review (receiver_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_review_sender_id_id   ON review (sender_id,   id DESC);

-- Функции пересчёта.
CREATE OR REPLACE FUNCTION recalc_user_review_aggregates(p_user_id bigint)
RETURNS void AS $$
BEGIN
    UPDATE "user"
    SET rating = COALESCE((
            SELECT AVG(rating)::numeric(3,2) FROM review WHERE receiver_id = p_user_id
        ), 0),
        reviews_count = (SELECT COUNT(*) FROM review WHERE receiver_id = p_user_id)
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trg_review_aggregates() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM recalc_user_review_aggregates(NEW.receiver_id);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM recalc_user_review_aggregates(OLD.receiver_id);
    ELSIF TG_OP = 'UPDATE' THEN
        PERFORM recalc_user_review_aggregates(NEW.receiver_id);
        IF NEW.receiver_id <> OLD.receiver_id THEN
            PERFORM recalc_user_review_aggregates(OLD.receiver_id);
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER review_aggregates_trigger
AFTER INSERT OR UPDATE OR DELETE ON review
FOR EACH ROW EXECUTE FUNCTION trg_review_aggregates();
