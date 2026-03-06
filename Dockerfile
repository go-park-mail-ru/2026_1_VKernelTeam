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
COPY static ./static
COPY .env .env

# Слушаем 8000 порт
EXPOSE 8000

# Запуск контейнера
ENTRYPOINT ["./clover"]
