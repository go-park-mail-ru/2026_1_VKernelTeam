ALTER TABLE support_ticket
    ADD COLUMN rating SMALLINT DEFAULT NULL
        CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5));
