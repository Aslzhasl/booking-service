# ---- Build stage ----
FROM golang:1.24 AS builder

WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарник
RUN go build -o app main.go

# ---- Final stage ----
FROM gcr.io/distroless/base-debian11

WORKDIR /app

COPY --from=builder /app/app /app/app

EXPOSE 8082

ENTRYPOINT ["/app/app"]
