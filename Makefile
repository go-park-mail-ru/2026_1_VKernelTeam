.PHONY: help run rebuild stop deploy test build build-auth build-support swag lint fmt vet clean proto proto-install \
       logs-clickhouse grafana-open status

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
	@echo "  logs-auth         - Показать логи auth-сервиса"
	@echo "  logs-support      - Показать логи support-сервиса"
	@echo ""
	@echo "  Сборка:"
	@echo "  build             - Собрать бинарник монолита"
	@echo "  build-auth        - Собрать бинарник auth-сервиса"
	@echo "  build-support     - Собрать бинарник support-сервиса"
	@echo "  proto             - Сгенерировать Go-код из proto-файлов"
	@echo "  swag              - Сгенерировать Swagger-документацию"
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

# Поднимает всё: общая БД, Redis, Kafka, монолит, auth, support, gateway, observability.
# Использует уже собранные образы. Если их нет — Compose соберёт автоматически.
# Не лезет в Docker Hub проверять метаданные базовых образов, так что работает офлайн,
# если контейнеры/кэш уже есть локально.
run:
	$(COMPOSE) up -d --remove-orphans

# Пересобирает образы (Go-код или Dockerfile поменялись) и поднимает стек.
# Требует доступ к Docker Hub для проверки базовых образов.
rebuild:
	$(COMPOSE) up -d --build --remove-orphans
	@echo ""
	@echo "╔══════════════════════════════════════════════════╗"
	@echo "║  Clover запущен                                 ║"
	@echo "║                                                 ║"
	@echo "║  Gateway:   http://localhost:$(GATEWAY_PORT)               ║"
	@echo "║  Монолит:   http://localhost:$(MONOLITH_PORT)               ║"
	@echo "║  Auth HTTP: http://localhost:$(AUTH_HTTP_PORT)               ║"
	@echo "║  Auth gRPC: localhost:$(AUTH_GRPC_PORT)                      ║"
	@echo "║  Support:   http://localhost:$(SUPPORT_HTTP_PORT)               ║"
	@echo "║  Kafka:     localhost:$(KAFKA_PORT)                      ║"
	@echo "║                                                 ║"
	@echo "║  Observability:                                 ║"
	@echo "║  Grafana:    http://localhost:$(GRAFANA_PORT)               ║"
	@echo "║  Prometheus: http://localhost:$(PROMETHEUS_PORT)               ║"
	@echo "║  ClickHouse: http://localhost:$(CLICKHOUSE_PORT)               ║"
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

logs-vector:
	$(COMPOSE) logs -f --tail=50 vector

# ─── Observability ────────────────────────────────────────────────────────────

logs-clickhouse:
	@curl -s "http://localhost:$(CLICKHOUSE_PORT)/?query=SELECT+timestamp,level,service,message,request_id+FROM+logs.service_logs+ORDER+BY+timestamp+DESC+LIMIT+25+FORMAT+PrettyCompact"

grafana-open:
	@open "http://localhost:$(GRAFANA_PORT)" 2>/dev/null || xdg-open "http://localhost:$(GRAFANA_PORT)" 2>/dev/null || echo "Grafana: http://localhost:$(GRAFANA_PORT)"

status:
	$(COMPOSE) ps

# Генерация документации Swagger
swag:
	swag init -g cmd/server/main.go -o ./api

# ─── Сборка ──────────────────────────────────────────────────────────────────

build: swag
	go build -o bin/clover ./cmd/server/main.go

build-auth:
	go build -o bin/auth-service ./services/auth/cmd/server/main.go

build-support:
	go build -o bin/support-service ./services/support/cmd/server/main.go

# ─── Proto ────────────────────────────────────────────────────────────────────

# Один раз: ставит компилятор protoc и Go-плагины.
# Без них `make proto` упадёт с "protoc: No such file or directory".
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
	@go test -coverprofile=coverage.tmp ./internal/... ./pkg/... ./services/... > /dev/null
	@grep -v -E "mocks|domain" coverage.tmp > coverage.out
	@go test -cover ./internal/... ./pkg/... ./services/... | grep -v -E "mocks|domain" | awk '{ \
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
	golangci-lint run --fix ./internal/... ./pkg/... ./cmd/... ./services/...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	go clean
	rm -f coverage.out coverage.html
	rm -rf bin/
