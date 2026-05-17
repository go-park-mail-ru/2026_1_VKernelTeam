-- Расширение должно быть пред-загружено через shared_preload_libraries
-- (см. DB/config/postgresql.custom.conf). Здесь делаем CREATE EXTENSION,
-- чтобы появилась view pg_stat_statements в БД clover.
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
