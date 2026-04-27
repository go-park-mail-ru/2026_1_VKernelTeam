.PHONY: help run deploy test test-verbose test-coverage test-auth test-storage build swag lint fmt vet clean migrate

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Разработка (локально):"
	@echo "  run               - Быстрый запуск локально (DB/Redis в Docker, сервер нативно с логами)"
	@echo "  swag              - Сгенерировать Swagger-документацию"
	@echo "  build             - Скомпилировать бинарный файл приложения"
	@echo ""
	@echo "  Тесты:"
	@echo "  test              - Запустить все тесты"
	@echo "  test-verbose      - Запустить все тесты с подробным выводом"
	@echo "  test-coverage     - Запустить тесты и показать отчет о покрытии"
	@echo "  test-auth         - Запустить тесты только для сервиса аутентификации"
	@echo "  test-storage      - Запустить тесты только для слоя хранения (storage)"
	@echo ""
	@echo "  Деплой (только на сервере):"
	@echo "  deploy            - Обновить код (git pull) и перезапустить все контейнеры"
	@echo ""
	@echo "  Утилиты:"
	@echo "  lint              - Запустить линтер (golangci-lint)"
	@echo "  fmt               - Отформатировать код"
	@echo "  vet               - Запустить go vet"
	@echo "  clean             - Удалить временные файлы и отчеты о покрытии"
	@echo "  migrate           - Применить миграции локально"

# ─── Разработка ───────────────────────────────────────────────────────────────

# Генерация документации Swagger
swag:
	swag init -g cmd/server/main.go -o ./api

# Быстрый локальный запуск: поднимает только зависимости в Docker, сервер — нативно
run: swag
	docker compose --env-file .env -f deployments/docker-compose.yaml up -d db db_migrate redis
	go run ./cmd/server/main.go --config=./config/local.json

# Сборка приложения в исполняемый файл
build: swag
	go build -o bin/clover ./cmd/server/main.go

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

# ─── Деплой (только на сервере) ───────────────────────────────────────────────

# Обновление кода и перезапуск всех контейнеров на сервере
deploy:
	sudo git pull
	docker compose --env-file .env -f deployments/docker-compose.yaml up -d --build --remove-orphans
	docker image prune -f

# ─── Утилиты ──────────────────────────────────────────────────────────────────

migrate:
	migrate -path ./internal/repository/postgres/migrations \
		-database "$(DATABASE_URL)" up

migrate-ads-img:
	go run ./cmd/migrate-images/

lint:
	golangci-lint run --fix ./internal/... ./pkg/... ./cmd/...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	go clean
	rm -f coverage.out coverage.html
	rm -rf bin/
