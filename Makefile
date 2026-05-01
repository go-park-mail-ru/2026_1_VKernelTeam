.PHONY: help run rebuild stop deploy test build-auth build-support build-catalog build-commerce lint fmt vet clean proto proto-install swag swag-install \
       logs logs-auth logs-support logs-catalog logs-commerce logs-vector logs-clickhouse grafana-open status

include .env
export

COMPOSE = docker compose --env-file .env -f deployments/docker-compose.yaml

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Разработка:"
	@echo "  run               - Поднять стек (использует кэшированные образы; сеть в Docker Hub не нужна)"
	@echo "  rebuild           - Пересобрать образы и поднять (после изменения Go-кода/Dockerfile)"
	@echo "  stop              - Остановить все контейнеры"
	@echo "  logs              - Показать логи всех сервисов"
	@echo "  logs-auth/-support/-catalog/-commerce - логи конкретного сервиса"
	@echo ""
	@echo "  Сборка:"
	@echo "  build-auth        - Собрать бинарник auth-сервиса"
	@echo "  build-support     - Собрать бинарник support-сервиса"
	@echo "  build-catalog     - Собрать бинарник catalog-сервиса"
	@echo "  build-commerce    - Собрать бинарник commerce-сервиса"
	@echo "  proto             - Сгенерировать Go-код из proto-файлов"
	@echo "  proto-install     - Установить protoc + Go-плагины (один раз)"
	@echo "  swag              - Перегенерировать api/swagger.{json,yaml} из аннотаций хендлеров"
	@echo "  swag-install      - Установить swag CLI (один раз)"
	@echo ""
	@echo "  Тесты:"
	@echo "  test              - Запустить тесты с покрытием"
	@echo ""
	@echo "  Деплой:"
	@echo "  deploy            - git pull + пересобрать контейнеры"
	@echo ""
	@echo "  Observability:"
	@echo "  logs-clickhouse   - Показать последние логи из ClickHouse"
	@echo "  grafana-open      - Открыть Grafana в браузере"
	@echo "  status            - Статус всех контейнеров"
	@echo ""
	@echo "  Утилиты:"
	@echo "  lint / fmt / vet / clean"

# ─── Разработка ───────────────────────────────────────────────────────────────

# Поднимает всё: общая БД, Redis, Kafka, auth, support, catalog, commerce, gateway, observability.
# Использует уже собранные образы. Если их нет — Compose соберёт автоматически.
# Не лезет в Docker Hub проверять метаданные базовых образов, так что работает офлайн,
# если контейнеры/кэш уже есть локально.
# Префикс swag — на случай, если забыли перегенерировать перед запуском.
run: swag
	$(COMPOSE) up -d --remove-orphans

# Пересобирает образы (Go-код или Dockerfile поменялись) и поднимает стек.
# Требует доступ к Docker Hub для проверки базовых образов.
rebuild: swag
	$(COMPOSE) up -d --build --remove-orphans
	@echo ""
	@echo "╔══════════════════════════════════════════════════╗"
	@echo "║  Clover запущен                                 ║"
	@echo "║                                                 ║"
	@echo "║  Gateway:    http://localhost:$(GATEWAY_PORT)              ║"
	@echo "║  Auth HTTP:  http://localhost:$(AUTH_HTTP_PORT)              ║"
	@echo "║  Auth gRPC:  localhost:$(AUTH_GRPC_PORT)                     ║"
	@echo "║  Support:    http://localhost:$(SUPPORT_HTTP_PORT)              ║"
	@echo "║  Catalog:    http://localhost:$(CATALOG_HTTP_PORT)              ║"
	@echo "║  Catalog gRPC: localhost:$(CATALOG_GRPC_PORT)                  ║"
	@echo "║  Commerce:   http://localhost:$(COMMERCE_HTTP_PORT)              ║"
	@echo "║  Swagger UI: http://localhost:$(GATEWAY_PORT)/swagger/      ║"
	@echo "║  Kafka:      localhost:$(KAFKA_PORT)                     ║"
	@echo "║                                                 ║"
	@echo "║  Observability:                                 ║"
	@echo "║  Grafana:    http://localhost:$(GRAFANA_PORT)              ║"
	@echo "║  Prometheus: http://localhost:$(PROMETHEUS_PORT)              ║"
	@echo "║  ClickHouse: http://localhost:$(CLICKHOUSE_PORT)              ║"
	@echo "║                                                 ║"
	@echo "║  make stop  — остановить всё                    ║"
	@echo "║  make logs  — посмотреть логи                   ║"
	@echo "╚══════════════════════════════════════════════════╝"

stop:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f --tail=50

logs-auth:
	$(COMPOSE) logs -f --tail=50 auth

logs-support:
	$(COMPOSE) logs -f --tail=50 support

logs-catalog:
	$(COMPOSE) logs -f --tail=50 catalog

logs-commerce:
	$(COMPOSE) logs -f --tail=50 commerce

logs-vector:
	$(COMPOSE) logs -f --tail=50 vector

# ─── Observability ────────────────────────────────────────────────────────────

logs-clickhouse:
	@curl -s "http://localhost:$(CLICKHOUSE_PORT)/?query=SELECT+timestamp,level,service,message,request_id+FROM+logs.service_logs+ORDER+BY+timestamp+DESC+LIMIT+25+FORMAT+PrettyCompact"

grafana-open:
	@open "http://localhost:$(GRAFANA_PORT)" 2>/dev/null || xdg-open "http://localhost:$(GRAFANA_PORT)" 2>/dev/null || echo "Grafana: http://localhost:$(GRAFANA_PORT)"

status:
	$(COMPOSE) ps

# ─── Сборка ──────────────────────────────────────────────────────────────────

build-auth:
	go build -o bin/auth-service ./services/auth/cmd/server/main.go

build-support:
	go build -o bin/support-service ./services/support/cmd/server/main.go

build-catalog:
	go build -o bin/catalog-service ./services/catalog/cmd/server/main.go

build-commerce:
	go build -o bin/commerce-service ./services/commerce/cmd/server/main.go

# ─── Swagger ──────────────────────────────────────────────────────────────────

# Генерация api/swagger.{json,yaml} из аннотаций @Summary/@Param/@Failure
# в services/<svc>/internal/delivery/handlers/. Глобальная мета (@title, @host,
# @securityDefinitions) лежит в api/swagger_info.go.
#
# Сейчас аннотированы catalog и commerce. Auth/support без аннотаций — их роуты
# в спеку не попадут, пока кто-нибудь не накидает @Summary. Это не блокер.
#
# Цель fail-safe: если swag CLI не установлен — пропускает с подсказкой;
# если генерация падает — логирует и оставляет старый api/swagger.yaml.
# `make run` и `make rebuild` зовут swag заранее, чтобы фронт получал свежий спек.
swag:
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g swagger_info.go --dir ./api,./services/auth,./services/support,./services/catalog,./services/commerce --parseInternal -o ./api 2>&1 \
			| grep -vE '^\s*$$|warning: failed to get package name' \
			| tail -5; \
	else \
		echo "[swag] CLI не установлен — пропускаю (один раз: make swag-install)"; \
	fi

swag-install:
	go install github.com/swaggo/swag/cmd/swag@latest

# ─── Proto ────────────────────────────────────────────────────────────────────

# Один раз: ставит компилятор protoc и Go-плагины.
proto-install:
	@which protoc >/dev/null 2>&1 || (echo "Installing protobuf-compiler (требуется sudo):" && sudo apt-get update && sudo apt-get install -y protobuf-compiler)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Готово. protoc + Go-плагины установлены."

proto:
	@which protoc >/dev/null 2>&1 || (echo "protoc не установлен. Запусти: make proto-install" && exit 1)
	protoc \
		--proto_path=proto \
		--go_out=proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
		auth/v1/auth.proto \
		catalog/v1/catalog.proto

# ─── Тесты ────────────────────────────────────────────────────────────────────

test:
	@go test -coverprofile=coverage.tmp ./pkg/... ./services/... > /dev/null
	@grep -v -E "mocks|domain" coverage.tmp > coverage.out
	@go test -cover ./pkg/... ./services/... | grep -v -E "mocks|domain" | awk '{ \
		if ($$1 == "ok") { \
			printf "%-100s %s\n", $$2, $$(NF-2) " " $$(NF-1) " " $$NF; \
		} else { \
			print $$0; \
		} \
	}'
	@echo "--------------------------------------------------------------------------------------------------------------------------"
	@go tool cover -func=coverage.out | grep total | awk '{printf "%-100s %s\n", "TOTAL PROJECT COVERAGE:", $$3}'
	@echo "--------------------------------------------------------------------------------------------------------------------------"
	@rm coverage.tmp coverage.out

# ─── Деплой ───────────────────────────────────────────────────────────────────

deploy:
	sudo git pull
	$(COMPOSE) up -d --build --remove-orphans
	docker image prune -f

# ─── Утилиты ──────────────────────────────────────────────────────────────────

lint:
	golangci-lint run --fix ./pkg/... ./services/...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	go clean
	rm -f coverage.out coverage.html
	rm -rf bin/
