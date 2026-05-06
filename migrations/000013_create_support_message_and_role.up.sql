CREATE TABLE support_message (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ticket_id  bigint NOT NULL REFERENCES support_ticket (id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES "user" (id),
    text       text NOT NULL CHECK (length(text) > 0),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_support_message_ticket_id ON support_message (ticket_id);

ALTER TABLE "user"
    ADD COLUMN role varchar(20) NOT NULL DEFAULT 'user'
        CHECK (role IN ('user', 'support', 'admin'));
