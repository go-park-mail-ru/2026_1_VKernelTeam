#!/usr/bin/env bash
# Прогон baseline: clean → 100k create → 10k read by-id → 100 list → 1k search.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

LOADER=./perf_test/bin/loader
OUT=./perf_test/results
mkdir -p "$OUT"

echo ">> rebuilding loader"
go build -o "$LOADER" ./perf_test/cmd/loader/

echo ">> cleaning DB"
./perf_test/scripts/clean_db.sh >/dev/null

TS=$(date +%Y%m%d_%H%M%S)
LABEL="${1:-baseline}"

echo ">> 1) CREATE 100k @ c=100"
"$LOADER" create \
  --total 100000 --c 100 \
  --out "$OUT/${LABEL}_${TS}_create.json" \
  2>&1 | tee "$OUT/${LABEL}_${TS}_create.log"

MAXID=$(docker exec -e PGPASSWORD=qwerty postgres psql -U postgres -d clover -tA -c "SELECT max(id) FROM product;")
echo "max product id = $MAXID"

echo ">> 2) READ by-id 10k @ c=100"
"$LOADER" read --mode by-id --max-id "$MAXID" \
  --total 10000 --c 100 \
  --out "$OUT/${LABEL}_${TS}_read_byid.json" \
  2>&1 | tee "$OUT/${LABEL}_${TS}_read_byid.log"

echo ">> 3) LIST /ads 100 @ c=10 (full-table scan)"
"$LOADER" read --mode list \
  --total 100 --c 10 \
  --out "$OUT/${LABEL}_${TS}_read_list.json" \
  2>&1 | tee "$OUT/${LABEL}_${TS}_read_list.log"

echo ">> 4) SEARCH 1k @ c=50"
"$LOADER" read --mode search --q "Объявление" \
  --total 1000 --c 50 \
  --out "$OUT/${LABEL}_${TS}_read_search.json" \
  2>&1 | tee "$OUT/${LABEL}_${TS}_read_search.log"

echo ">> done. results in $OUT (label=$LABEL ts=$TS)"
