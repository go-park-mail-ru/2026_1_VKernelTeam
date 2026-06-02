#!/usr/bin/env bash
# Поднимает стек, необходимый для нагрузочного теста.
# Запускать из корня репозитория: ./perf_test/scripts/up.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

docker compose --env-file ./.env -f deployments/docker-compose.yaml up -d db redis kafka
docker compose --env-file ./.env -f deployments/docker-compose.yaml run --rm db_migrate >/dev/null
docker compose --env-file ./.env -f deployments/docker-compose.yaml up -d --build auth catalog

echo "Waiting for catalog HTTP..."
for i in {1..60}; do
  if curl -sf http://localhost:8004/api/v1/ads >/dev/null 2>&1; then
    echo "catalog is up"
    exit 0
  fi
  sleep 1
done
echo "catalog did not become ready in time" >&2
exit 1
