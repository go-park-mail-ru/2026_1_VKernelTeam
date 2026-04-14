FROM golang:1.26.0-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем и загружаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем код проекта
COPY . .

# Собираем бинарь без си функций
RUN CGO_ENABLED=0 GOOS=linux go build -o clover ./cmd/server/main.go

# Образ линукс
FROM alpine:3.20

# Устанавливаем рабочую директорию
WORKDIR /app

COPY --from=builder /app/clover .
COPY config ./config
COPY .env .env
COPY api ./api

# Создаём пустые папки для статики и хранилища
RUN mkdir -p /app/static /app/storage

# Слушаем 8000 порт
EXPOSE 8000

# Запуск контейнера
ENTRYPOINT ["./clover"]
