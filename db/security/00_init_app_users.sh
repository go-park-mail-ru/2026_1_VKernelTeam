#!/usr/bin/env bash
# ============================================================================
#  Инициализация сервисных пользователей PostgreSQL.
# ----------------------------------------------------------------------------
#  Что делает:
#    Подхватывает пароли сервисных учёток из переменных окружения и
#    передаёт их в 01_create_app_users.sql через psql -v. SQL-скрипт уже
#    содержит всю логику создания ролей и грантов — здесь только
#    проброс переменных.
#
#  Почему .sh, а не чистый .sql:
#    Образ postgres:17-alpine исполняет содержимое
#    /docker-entrypoint-initdb.d/ при первой инициализации тома, но
#    psql без обёртки не умеет читать переменные окружения. Поэтому
#    .sh — единственный безопасный способ не зашивать пароль в SQL.
#
#  Идемпотентность:
#    Скрипт запускается ровно один раз при создании volume `pgdata`.
#    Для повторной инициализации — удалить том (`docker compose down -v`).
#    SQL-часть сама идемпотентна (см. DO-блоки в 01_create_app_users.sql).
# ============================================================================
set -euo pipefail

# Обязательные переменные.
: "${AUTH_DB_PASSWORD:?AUTH_DB_PASSWORD env variable is required}"
: "${CATALOG_DB_PASSWORD:?CATALOG_DB_PASSWORD env variable is required}"
: "${COMMERCE_DB_PASSWORD:?COMMERCE_DB_PASSWORD env variable is required}"
: "${SUPPORT_DB_PASSWORD:?SUPPORT_DB_PASSWORD env variable is required}"
: "${MONITORING_DB_PASSWORD:?MONITORING_DB_PASSWORD env variable is required}"

# В docker-entrypoint-initdb.d суперпользователь postgres уже создан
# и переменная POSTGRES_DB указывает на целевую БД.
psql \
    --username "${POSTGRES_USER}" \
    --dbname   "${POSTGRES_DB}" \
    --set ON_ERROR_STOP=on \
    --set "auth_pass=${AUTH_DB_PASSWORD}" \
    --set "catalog_pass=${CATALOG_DB_PASSWORD}" \
    --set "commerce_pass=${COMMERCE_DB_PASSWORD}" \
    --set "support_pass=${SUPPORT_DB_PASSWORD}" \
    --set "monitoring_pass=${MONITORING_DB_PASSWORD}" \
    --file /sql/01_create_app_users.sql

echo "[init] Service users created successfully."
