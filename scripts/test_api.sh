#!/bin/bash
# Скрипт для тестирования API через curl
# Использование: ./scripts/test_api.sh

BASE="http://localhost:8000/api/v1"
COOKIE_FILE="/tmp/clover_cookies.txt"

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

print_result() {
    local status=$1
    local body=$2
    if [ "$status" -ge 200 ] && [ "$status" -lt 300 ]; then
        echo -e "  Status: ${GREEN}${status}${NC}"
    else
        echo -e "  Status: ${RED}${status}${NC}"
    fi
    echo "  Body: $body"
}

# ═══════════════════════════════════════════
# 1. GET /ads — без авторизации (публичная)
# ═══════════════════════════════════════════
print_header "1. GET /ads (публичная)"
RESP=$(curl -s -w "\n%{http_code}" "$BASE/ads")
BODY=$(echo "$RESP" | head -1 | python3 -c "import json,sys; ads=json.load(sys.stdin); print(f'{len(ads)} объявлений')" 2>/dev/null || echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

# ═══════════════════════════════════════════
# 2. POST /ads — без авторизации (должен 401)
# ═══════════════════════════════════════════
print_header "2. POST /ads без авторизации (ожидается 401)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/ads" \
    -H "Content-Type: application/json" \
    -d '{"category_id":1,"title":"Test Title","description":"Test description long","price":1000,"status":"active","photos":[]}')
BODY=$(echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

# ═══════════════════════════════════════════
# 3. Логин — получаем JWT + CSRF
# ═══════════════════════════════════════════
print_header "3. POST /auth/login"
LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@test.com","password":"Password123"}' \
    -c "$COOKIE_FILE")
LOGIN_BODY=$(echo "$LOGIN_RESP" | head -1)
LOGIN_STATUS=$(echo "$LOGIN_RESP" | tail -1)
print_result "$LOGIN_STATUS" "$LOGIN_BODY"

# Извлекаем CSRF из ответа
CSRF=$(echo "$LOGIN_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['csrf_token'])" 2>/dev/null)
echo -e "  CSRF Token: ${GREEN}${CSRF}${NC}"

# ═══════════════════════════════════════════
# 4. POST /ads — с авторизацией, без CSRF (должен 400)
# ═══════════════════════════════════════════
print_header "4. POST /ads с JWT, без CSRF (ожидается 400)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/ads" \
    -H "Content-Type: application/json" \
    -b "$COOKIE_FILE" \
    -d '{"user_id":1,"category_id":1,"title":"Test Title","description":"Test description long","price":1000,"status":"active","photos":[]}')
BODY=$(echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

# ═══════════════════════════════════════════
# 5. POST /ads — с JWT + CSRF, невалидные данные
# ═══════════════════════════════════════════
print_header "5. POST /ads — невалидные данные (title < 5 символов)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/ads" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $CSRF" \
    -b "$COOKIE_FILE" \
    -d '{"category_id":1,"title":"Hi","description":"Short","price":-5,"status":"wrong","photos":[]}')
BODY=$(echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

# ═══════════════════════════════════════════
# 6. POST /ads — всё корректно, создаём объявление
# ═══════════════════════════════════════════
print_header "6. POST /ads — создание объявления (ожидается 200)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/ads" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $CSRF" \
    -b "$COOKIE_FILE" \
    -c "$COOKIE_FILE" \
    -d '{
        "category_id": 3,
        "title": "Велосипед горный Trek",
        "description": "Отличный велосипед, практически новый, катались пару раз",
        "price": 45000,
        "status": "active",
        "photos": []
    }')
BODY=$(echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

# Обновляем CSRF из новой куки (сервер обновляет после POST)
NEW_CSRF=$(grep csrf_token "$COOKIE_FILE" | awk '{print $NF}')
echo -e "  Новый CSRF: ${GREEN}${NEW_CSRF}${NC}"

# ═══════════════════════════════════════════
# 7. Logout
# ═══════════════════════════════════════════
print_header "7. POST /auth/logout"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/auth/logout" \
    -b "$COOKIE_FILE")
BODY=$(echo "$RESP" | head -1)
STATUS=$(echo "$RESP" | tail -1)
print_result "$STATUS" "$BODY"

echo ""
echo -e "${CYAN}━━━ Тестирование завершено ━━━${NC}"
