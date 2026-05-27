-- Источник заказа и связь с чатом подтверждения покупки.
-- Нужны для GET /api/v1/profile/purchases: фронт показывает источник ('cart'/'chat')
-- и при source='chat' даёт ссылку открыть исходный чат.

ALTER TABLE "order" ADD COLUMN source text;
ALTER TABLE "order" ADD COLUMN chat_id bigint REFERENCES chat (id) ON DELETE SET NULL;

-- Бэкфилл: до этой миграции заказы создавались только из подтверждения покупки в чате
-- (cart checkout как отдельный сценарий ещё не реализован). Поэтому всем имеющимся
-- заказам ставим source='chat' и подбираем chat_id по (product_id, buyer_id).
UPDATE "order" o
SET source  = 'chat',
    chat_id = (
        SELECT c.id
        FROM order_item oi
        JOIN chat c
          ON c.product_id = oi.product_id
         AND c.buyer_id   = o.buyer_id
        WHERE oi.order_id = o.id
        ORDER BY c.id ASC
        LIMIT 1
    )
WHERE source IS NULL;

-- Остальные (без матчинг-чата по какой-то причине) — пометим как 'cart'
-- чтобы NOT NULL не падал.
UPDATE "order" SET source = 'cart' WHERE source IS NULL;

ALTER TABLE "order" ALTER COLUMN source SET NOT NULL;
ALTER TABLE "order" ADD CONSTRAINT order_source_check CHECK (source IN ('cart', 'chat'));

-- Курсорная пагинация GET /profile/purchases — по buyer_id DESC.
CREATE INDEX IF NOT EXISTS idx_order_buyer_id_desc ON "order" (buyer_id, id DESC);
