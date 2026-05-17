ALTER TABLE message ADD COLUMN msg_type text DEFAULT 'text';
-- 'text' для обычных, 'order' для системных уведомлений
