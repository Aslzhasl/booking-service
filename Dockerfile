# ────────────────────────────────────────────────────────────────────────────
# booking-service/Dockerfile
# ────────────────────────────────────────────────────────────────────────────

# 1) Build stage: собираем статический Go-бинарь
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Сборка вашего booking-service в единый бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service .

# ────────────────────────────────────────────────────────────────────────────
# 2) Runtime stage: минимальный Alpine для запуска
FROM alpine:latest

# Устанавливаем сертификаты для HTTPS (если ваш сервис делает внешние запросы по HTTPS)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Копируем собранный ранее бинарь
COPY --from=builder /app/booking-service .

# Cloud Run автоматически передаёт внутрь контейнера PORT=8080,
# но ваш код ожидает сначала HTTP_PORT, поэтому назначаем обе переменные:
ENV PORT=8080
ENV HTTP_PORT=8080

# Экспонируем 8080 — Cloud Run будет подключаться именно к нему
EXPOSE 8080

# По умолчанию вы указывали настройки БД через ENV в Dockerfile,
# но при деплое лучше задавать DATABASE_URL или отдельные переменные через --set-env-vars.
# Оставим их здесь, но они будут перекрыты командой gcloud run deploy.
ENV DB_HOST=34.56.50.46
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=1234
ENV DB_NAME=Booking-service
ENV DB_SSLMODE=disable

# Ссылки на внешние сервисы (можно оставить или задать через --set-env-vars)

ENV USER_SERVICE_URL=https://user-service-721348598691.europe-central2.run.app/api
ENV LISTING_SERVICE_URL=https://listing-service-721348598691.europe-central2.run.app/api

# Секрет для JWT (лучше передавать через Secret Manager или --set-env-vars)
ENV JWT_SECRET="verylongrandomstringyouwritehere-and-never-commit-an-obvious-password"

# Запускаем приложение: ваш main.go прочитает HTTP_PORT=8080 или PORT=8080
CMD ["./booking-service"]
