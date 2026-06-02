#!/usr/bin/env bash
# Полная очистка таблиц, связанных с продуктами. Категории/пользователей сохраняем.
set -euo pipefail
docker exec -e PGPASSWORD=qwerty postgres psql -U postgres -d clover -v ON_ERROR_STOP=1 -c "
  TRUNCATE
    product,
    product_image,
    product_status_history,
    product_price_history,
    favorite,
    product_view,
    product_characteristic,
    product_custom_characteristic,
    cart_item,
    chat,
    message,
    review,
    promotion,
    \"order\",
    order_item
  RESTART IDENTITY CASCADE;
"
