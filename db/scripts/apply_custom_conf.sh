#!/usr/bin/env bash
# ============================================================================
#  Подключение db/config/postgresql.custom.conf к основному postgresql.conf.
# ----------------------------------------------------------------------------
#  Контекст:
#    Образ postgres:17-alpine хранит свой основной postgresql.conf в
#    $PGDATA (по умолчанию /var/lib/postgresql/data). Чтобы кастомные
#    параметры применились, нужно либо подменить весь файл, либо
#    добавить include-директиву. Подмену делать не хочется (туда же
#    включаются параметры, заданные через POSTGRES_* env), поэтому
#    мы дописываем одну строку include и подкладываем кастомный файл
#    рядом.
#
#  Как запускается:
#    Положен в /docker-entrypoint-initdb.d/ — postgres исполняет .sh из
#    этой папки при первой инициализации тома (до того, как сервер
#    начнёт принимать клиентов).
# ============================================================================
set -euo pipefail

CONF_DIR="$PGDATA"
CONF_FILE="$CONF_DIR/postgresql.conf"
CUSTOM_SRC="/etc/postgresql/clover/postgresql.custom.conf"
HBA_SRC="/etc/postgresql/clover/pg_hba.conf"
INCLUDE_LINE="include = '/etc/postgresql/clover/postgresql.custom.conf'"

# 1. Каталог для логов pgbadger. Скрипт исполняется уже под пользователем
# postgres (не root), поэтому chown не нужен — наш volume на /var/log/postgresql
# смонтирован с правильным владельцем.
mkdir -p /var/log/postgresql || true

# 2. Подключаем include в postgresql.conf (один раз).
if ! grep -Fxq "$INCLUDE_LINE" "$CONF_FILE"; then
    echo "" >> "$CONF_FILE"
    echo "# --- Кастомная конфигурация Clover (см. db/config/) ---" >> "$CONF_FILE"
    echo "$INCLUDE_LINE" >> "$CONF_FILE"
fi

# 3. Подменяем pg_hba.conf: дефолтный 'trust all' оставлять опасно.
if [ -f "$HBA_SRC" ]; then
    install -m 0600 -o postgres -g postgres "$HBA_SRC" "$CONF_DIR/pg_hba.conf"
fi

echo "[apply_custom_conf] Custom postgresql.conf and pg_hba.conf installed."
