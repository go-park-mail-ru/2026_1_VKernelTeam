## Аргументация выбранных типов данных и ограничений целостности
- **Идентификаторы (PK/FK):** Используется `bigint GENERATED ALWAYS AS IDENTITY` вместо устаревшего `serial`. Это предотвращает ручную вставку дублирующихся ID и обеспечивает строгую автоинкрементацию.
- **Строки:** Везде используется тип `text`. В PostgreSQL он работает так же быстро, как и `varchar(n)`, но для защиты данных добавлены жесткие ограничения `CHECK (length(field) BETWEEN X AND Y)`. Поле email снабжено регулярным выражением для проверки формата.
- **Цены (price):** Используется `bigint` для хранения цены в минимальных единицах валюты (копейках). Это исключает ошибки округления типов с плавающей точкой (float/numeric). Добавлено ограничение `CHECK (price >= 0)`.
- **Даты (created_at, updated_at):** Используется `timestamp with time zone` (в DDL пишется как `timestamptz`). Это гарантирует корректную работу со временем независимо от часового пояса сервера.
- **Статусы:** Использованы текстовые поля со строгими проверками `CHECK (status IN (...))`, что заменяет ENUM и упрощает миграции при добавлении новых статусов.

## Ограничения (Constraints)
Во всех таблицах явно прописаны `PRIMARY KEY`. Применены `FOREIGN KEY` с каскадными операциями `ON DELETE CASCADE` там, где удаление родительской сущности должно безусловно удалять зависимые (например, сообщения в чате, просмотры товара). В критичных местах (оформленные заказы, категории) используется `ON DELETE RESTRICT` или `ON DELETE SET NULL` для защиты исторических данных.

## Функциональные зависимости:
### Relation user:
- {id} -> email, password_hash, first_name, second_name, avatar_path, rating, created_at, updated_at
- {email} -> id, password_hash, first_name, second_name, avatar_path, rating, created_at, updated_at

### Relation product:
- {id} -> seller_id, category_id, title, description, price, status, created_at, updated_at, deleted_at

### Relation order:
- {id} -> buyer_id, total_amount, status, created_at, updated_at

### Relation order_item:
- {order_id, product_id} -> price_at_purchase

### Relation category:
- {id} -> name, parent_id, created_at, updated_at
- {name} -> id, parent_id, created_at, updated_at

### Relation product_image:
- {id} -> product_id, file_path, sort_order, created_at, updated_at

### Relation favorite:
- {user_id, product_id} -> created_at

### Relation cart_item:
- {user_id, product_id} -> quantity, created_at, updated_at

### Relation chat:
- {id} -> product_id, buyer_id, seller_id, created_at, updated_at

### Relation message:
- {id} -> chat_id, sender_id, text_content, is_read, created_at, updated_at

## Доказательство соответствия нормальным формам:

### 1НФ (Первая нормальная форма)
Во всех отношениях атрибуты атомарны. Отсутствуют массивы (`text[]`) или `jsonb` структуры для хранения множественных значений (изображения вынесены в `product_image`). У каждой таблицы определен первичный ключ.

### 2НФ (Вторая нормальная форма)
Схема находится в 1НФ, и в ней нет частичных зависимостей от составных ключей. В таблицах `favorite`, `cart_item` и `order_item` составной первичный ключ. Все неключевые атрибуты в них (например, `quantity` в корзине или `price_at_purchase` в позициях заказа) зависят от всего составного ключа целиком.

### 3НФ и НФБК (Осознанная денормализация)
В архитектуре применены преднамеренные отступления от строгой 3НФ/НФБК ради оптимизации производительности под высокие нагрузки (Highload):
1. **Таблица `chat`:** Хранит атрибут `seller_id`, который транзитивно зависит от ключа через `product_id` ($Chat \rightarrow Product \rightarrow Seller$). Эта денормализация сделана намеренно, чтобы при запросе списка диалогов пользователя избежать ресурсоемкого `JOIN` с таблицей `product`.
2. **Таблица `user`:** Содержит агрегированное вычисляемое поле `rating`. По правилам 3НФ его следовало бы вычислять динамически `AVG(rating)` из таблицы `review`. Однако для профилей с тысячами отзывов это создаст критическую нагрузку на БД при каждом просмотре. Значение предрассчитано.

В остальном схема строго соответствует 3НФ и НФБК (детерминантами выступают только потенциальные ключи).
