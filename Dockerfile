# booking-service/Dockerfile

# 1) Сборочный этап: используем golang:1.24-alpine для загрузки зависимостей
#    и компиляции одного статически слинкованного бинарника.
FROM golang:1.24-alpine AS builder

# Если в программе есть зависимости из Git, понадобится git
RUN apk add --no-cache git

WORKDIR /app

# Копируем go.mod и go.sum, чтобы подтянуть модули
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код в /app
COPY . .

# Собираем бинарь из main.go, поскольку ваш main.go лежит прямо в корне
RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service .

# ──────────────────────────────────────────────────────────────────────────────
# 2) Финальный «легковесный» образ
# ──────────────────────────────────────────────────────────────────────────────
FROM alpine:latest

# Для работы HTTPS-запросов (User/Device Service по HTTPS) нужны CA-сертификаты
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Копируем собранный бинарь из предыдущего этапа
COPY --from=builder /app/booking-service .

# Экспонируем порт, на котором Booking Service слушает (HTTP_PORT)
EXPOSE 8082

# По умолчанию задаём переменные окружения (можно переопределять в docker-compose или при docker run)
ENV HTTP_PORT=8082
ENV DB_HOST=34.45.98.93
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=1234
ENV DB_NAME=Booking-service
ENV DB_SSLMODE=disable

# URL-ы развернутых сервисов (User и Device/Listing)
ENV USER_SERVICE_URL=https://user-service-387629641329.us-central1.run.app/api
ENV LISTING_SERVICE_URL=https://device-service-387629641329.us-central1.run.app/api

# Секрет JWT (должен совпадать с тем, что в User Service)
ENV JWT_SECRET=verylongrandomstringyouwritehere-and-never-commit-an-obvious-password

# Точка входа: запускаем наш скомпилированный бинарь
CMD ["./booking-service"]
