CREATE TABLE support_ticket (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES "user"(id),
    category    VARCHAR(50) NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'open',
    title       VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);
