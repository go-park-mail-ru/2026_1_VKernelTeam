.PHONY: help run stop deploy test build build-auth swag lint fmt vet clean proto

COMPOSE = docker compose --env-file .env -f deployments/docker-compose.yaml

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Разработка:"
	@echo "  run               - Поднять всю инфраструктуру (монолит + auth + gateway + kafka)"
	@echo "  stop              - Остановить все контейнеры"
	@echo "  logs              - Показать логи всех сервисов"
	@echo "  logs-auth         - Показать логи auth-сервиса"
	@echo ""
	@echo "  Сборка:"
	@echo "  build             - Собрать бинарник монолита"
	@echo "  build-auth        - Собрать бинарник auth-сервиса"
	@echo "  proto             - Сгенерировать Go-код из proto-файлов"
	@echo "  swag              - Сгенерировать Swagger-документацию"
	@echo ""
	@echo "  Тесты:"
	@echo "  test              - Запустить тесты с покрытием"
	@echo ""
	@echo "  Деплой:"
	@echo "  deploy            - git pull + пересобрать контейнеры"
	@echo ""
	@echo "  Утилиты:"
	@echo "  lint / fmt / vet / clean"

# ─── Разработка ───────────────────────────────────────────────────────────────

# Поднимает всё: БД монолита, БД auth, Redis, Kafka, монолит, auth, gateway
run:
	$(COMPOSE) up -d --build --remove-orphans
	@echo ""
	@echo "╔══════════════════════════════════════════════════╗"
	@echo "║  Clover запущен                                 ║"
	@echo "║                                                 ║"
	@echo "║  Gateway:   http://localhost                    ║"
	@echo "║  Монолит:   http://localhost:8000               ║"
	@echo "║  Auth HTTP: http://localhost:8001               ║"
	@echo "║  Auth gRPC: localhost:9001                      ║"
	@echo "║  Kafka:     localhost:9092                      ║"
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

# Генерация документации Swagger
swag:
	swag init -g cmd/server/main.go -o ./api

# ─── Сборка ──────────────────────────────────────────────────────────────────

build: swag
	go build -o bin/clover ./cmd/server/main.go

build-auth:
	go build -o bin/auth-service ./services/auth/cmd/server/main.go

# ─── Proto ────────────────────────────────────────────────────────────────────

proto:
	protoc \
		--proto_path=proto \
		--go_out=proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
		auth/v1/auth.proto

# ─── Тесты ────────────────────────────────────────────────────────────────────

test:
	go test ./...

test-verbose:
	go test -v ./...

test-auth:
	go test -v ./internal/usecase/auth/...

test-coverage:
	@go test -coverprofile=coverage.tmp ./internal/... ./pkg/... > /dev/null
	@grep -v -E "mocks|domain" coverage.tmp > coverage.out
	@go test -cover ./internal/... ./pkg/... | grep -v -E "mocks|domain" | awk '{ \
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
