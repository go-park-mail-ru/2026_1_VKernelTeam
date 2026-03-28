.PHONY: help test test-verbose test-coverage test-auth test-storage test-watch clean
#.DEFAULT_GOAL:=run

help:
	@echo "Available targets:"
	@echo "  test              - Запустить все тесты"
	@echo "  test-verbose      - Запустить все тесты с подробным выводом"
	@echo "  test-coverage     - Запустить тесты и показать отчет о покрытии"
	@echo "  test-auth         - Запустить тесты только для сервиса аутентификации"
	@echo "  test-storage      - Запустить тесты только для слоя хранения (storage)"
	@echo "  clean             - Удалить временные файлы и отчеты о покрытии"
	@echo "  run               - Запустить приложение локально"
	@echo "  build             - Скомпилировать бинарный файл приложения"
	@echo "  swag              - Сгенерировать Swagger-документацию"
	@echo "  deploy            - Обновить код и перезапустить контейнеры на сервере"


migrate:
	migrate -path ./internal/repository/postgres/migrations \
		-database "postgres://postgres:qwerty@localhost:5432/clover?sslmode=disable" up

# Генерация документации Swagger
swag:
	swag init -g cmd/server/main.go -o ./api

# Запуск приложения с локальным конфигом
run: swag
	docker compose --env-file .env -f deployments/docker-compose.yaml up -d db db_migrate redis
	go run ./cmd/server/main.go --config=./config/local.json

# Запуск всех тестов
test:
	go test ./...

# Запуск тестов с подробным выводом
test-verbose:
	go test -v ./...

# Проверка покрытия кода тестами
test-coverage:
	go test -cover ./internal/... ./pkg/...

# Тестирование только логики аутентификации
test-auth:
	go test -v ./internal/usecase/auth/...

# Тестирование только компонентов хранилища
test-storage:
	go test -v ./internal/repository/...

# Сборка приложения в исполняемый файл
build: swag
	go build -o bin/clover ./cmd/server/main.go

# Очистка проекта от собранных файлов и отчетов
clean:
	go clean
	rm -f coverage.out coverage.html
	rm -rf bin/

# Запуск линтера (требуется установленный golangci-lint)
lint:
	golangci-lint run --fix ./internal/... ./pkg/... ./cmd/...

# Форматирование кода по стандарту Go
fmt:
	go fmt ./...

# Запуск статического анализатора go vet
vet:
	go vet ./...

# Обновление и перезапуск на сервере
# Используется zero-downtime подход: сначала сборка, затем замена контейнеров
deploy:
	sudo git pull
	docker compose --env-file .env -f deployments/docker-compose.yaml up -d --build --remove-orphans
	docker image prune -f

# Полный перезапуск: стоп, генерация доки и чистый старт
restart: stop swag run

# Остановить только приложение, если оно в докере (или просто прибраться)
stop:
	docker compose -f deployments/docker-compose.yaml stop
