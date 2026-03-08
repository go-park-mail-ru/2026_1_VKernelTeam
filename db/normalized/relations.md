## Аргументация выбранных типов данных и ограничений целостности
- **Идентификаторы (PK/FK):** Мы используем bigint GENERATED ALWAYS AS IDENTITY вместо запрещенного serial. Это современный стандарт PostgreSQL, который жестко привязывает последовательность к таблице и предотвращает ручную вставку дублирующихся ID.

- **Строки:** Везде используется тип text, который в PostgreSQL работает так же быстро, как и varchar, но не требует жесткого указания длины.

- **Цены (price):** Используется bigint для хранения цены в рублях. Это исключает ошибки округления, свойственные типам с плавающей точкой (float/numeric).

- **Даты (created_at, updated_at):** Используется timestamp with time zone (в DDL пишется как timestamptz). Это гарантирует, что мы всегда знаем точное время события независимо от часового пояса сервера. Обязательное наличие этих полей учтено.


## Ограничения (Constraints)
Во всех таблицах явно прописаны PRIMARY KEY. Названия в единственном числе. Применены FOREIGN KEY с каскадными операциями ON DELETE CASCADE там, где удаление родительской сущности должно удалять зависимые (например, сообщения в чате). Для защиты от неполных данных везде, где возможно, указан NOT NULL.


## Описание таблиц (Отношения)
- **user:** Хранит данные профиля пользователя.

- **product:** Основная сущность объявления/товара. Содержит ссылки на продавца и категорию.

- **product_price_history:** Лог изменения цен на товары для истории.

- **product_status_history:** Лог изменения статусов (активен, продан, заблокирован) товара.

- **category:** Иерархический справочник категорий товаров.

- **product_image:** Хранит ссылки на изображения товаров.

- **favorite:** Таблица-связка для хранения избранных товаров пользователя (составной ключ).

- **cart_item:** Таблица-связка для корзины покупок (составной ключ).

- **review:** Отзывы покупателей о продавцах (или товарах).

- **chat:** Сущность чата между продавцом и покупателем по конкретному товару.

- **message:** Сообщения, привязанные к конкретному чату.

## Функциональные зависимости:
### Relation user:
- {id} -> email, password_hash, first_name, second_name, avatar_url, rating, created_at, updated_at
- {email} -> id, password_hash, first_name, second_name, avatar_url, rating, created_at, updated_at

### Relation product:
- {id} -> seller_id, category_id, title, description, price, status, views_count, favorites_count, created_at, updated_at, deleted_at

### Relation product_price_history:
- {id} -> product_id, old_price, new_price, created_at, updated_at

### Relation product_status_history:
- {id} -> product_id, old_status, new_status, created_at, updated_at

### Relation category:
- {id} -> name, parent_id, created_at, updated_at
{name} -> id, parent_id, created_at, updated_at

### Relation product_image:
- {id} -> product_id, url, is_main, created_at, updated_at

### Relation favorite:
- {user_id, product_id} -> created_at, updated_at

### Relation cart_item:
- {user_id, product_id} -> created_at, updated_at

### Relation review:
- {id} -> sender_id, receiver_id, product_id, rating, content, created_at, updated_at

### Relation chat:
- {id} -> product_id, buyer_id, seller_id, created_at, updated_at

### Relation message:
- {id} -> chat_id, sender_id, text_content, is_read, created_at, updated_at

## Доказательство соответствия нормальным формам (1НФ, 2НФ, 3НФ, НФБК):
### 1НФ (Первая нормальная форма)
Во всех отношениях все атрибуты атомарны (неделимы). Не используем составные типы (json, массивы).

### 2НФ (Вторая нормальная форма)
Схема находится в 1НФ, и в ней нет частичных зависимостей. Для таблиц с простым первичным ключом (id) это выполняется автоматически. В таблицах favorite и cart_item с составным первичным ключом {user_id, product_id} неключевые атрибуты (created_at, updated_at) зависят только от всего составного ключа целиком.

### 3НФ (Третья нормальная форма)
Схема находится во 2НФ и не содержит транзитивных зависимостей. Все неключевые атрибуты зависят исключительно от первичного ключа, а не от других неключевых атрибутов. (Например, в product атрибут category_id хранит только ссылку, а название категории вынесено в отдельную таблицу category).

### НФБК (Нормальная форма Бойса-Кодда)
Схема находится в 3НФ. В наших отношениях детерминантами выступают только {id}, {email}, {name} (для категории) и комбинация {user_id, product_id}. Все они являются либо первичными ключами (PK), либо уникальными ключами (UK), следовательно, требования НФБК строго соблюдены.
