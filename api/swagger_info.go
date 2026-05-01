// Package api содержит общие константы (ApiPrefix) и swag-метаинформацию,
// по которой `swag init` собирает api/swagger.yaml + swagger.json из аннотаций
// в хендлерах всех сервисов (auth, support, catalog, commerce).
//
// Файл намеренно минимальный — настоящие @Summary/@Param/@Failure живут на
// конкретных HTTP-обработчиках в services/<name>/internal/delivery/handlers/.
package api

// @title           Clover API
// @version         1.0
// @description     API маркетплейса Клевер (микросервисная архитектура).
// @description     - Auth Service       (HTTP :8001, gRPC :9001) — авторизация, профиль, пользователи
// @description     - Support Service    (HTTP :8002)             — техподдержка
// @description     - Catalog Service    (HTTP :8004, gRPC :9004) — объявления, категории, избранное, просмотры
// @description     - Commerce Service   (HTTP :8006)             — корзина, чаты, заказы
// @description     Все запросы идут через API Gateway (nginx :8080).

// @host       clover-go.ru
// @BasePath   /api/v1

// @securityDefinitions.apikey CookieAuth
// @in   cookie
// @name token
// @description JWT session token

// @securityDefinitions.apikey CsrfCookieAuth
// @in   cookie
// @name csrf_token
// @description CSRF token stored in cookies

// @securityDefinitions.apikey CsrfHeaderAuth
// @in   header
// @name X-CSRF-Token
// @description CSRF token required in header
