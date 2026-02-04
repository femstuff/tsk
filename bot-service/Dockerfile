# Go бот Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o telegram-bot ./cmd

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/telegram-bot .
COPY --from=builder /app/.env ./.env

# Создаем папку для загрузок
RUN mkdir -p /downloads

EXPOSE 8080

CMD ["./telegram-bot"]
