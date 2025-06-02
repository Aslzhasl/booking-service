# booking-service/Dockerfile

# ────────────────────────────────────────────────────────────────────────────
# Сборочный этап
# ────────────────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service .

# ────────────────────────────────────────────────────────────────────────────
# Финальный стройный образ
# ────────────────────────────────────────────────────────────────────────────
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/booking-service .
# Экспонируем 8080, так как Cloud Run будет дозапускать прослушивание на PORT=8080
EXPOSE 8080

# Локально: HTTP_PORT=8082, но в Cloud Run - перезапишется PORT=8080

# Параметры подключения к БД и внешним сервисам
ENV DB_HOST=34.45.98.93
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=1234
ENV DB_NAME=Booking-service
ENV DB_SSLMODE=disable

ENV USER_SERVICE_URL=https://user-service-387629641329.us-central1.run.app/api
ENV LISTING_SERVICE_URL=https://device-service-387629641329.us-central1.run.app/api

ENV JWT_SECRET=verylongrandomstringyouwritehere-and-never-commit-an-obvious-password

CMD ["./booking-service"]
