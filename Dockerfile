# Стадия сборки
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Копируем go mod файлы (go.sum может отсутствовать, это нормально)
COPY go.mod ./
# Копируем go.sum если он существует (опционально)
COPY go.sum* ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app

# Стадия запуска
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из стадии сборки
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]