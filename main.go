package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"booking-service/internal/config"
	"booking-service/internal/handler"
	"booking-service/internal/middleware"
	"booking-service/internal/repository"
	"booking-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Сначала грузим .env, чтобы переменные окружения были доступны внутри LoadConfig()
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or could not be loaded, relying on real environment variables")
	}

	// 2. Теперь загружаем конфигурацию (она читает из os.Getenv)
	cfg := config.LoadConfig()

	// 3. Подключаемся к Postgres
	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	// Настройка пула соединений
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("✅ Connected to Postgres")

	// 4. Инициализация слоёв: репозиторий → сервис → handler
	bookingRepo := repository.NewBookingRepository(db)
	bookingSvc := service.NewBookingService(
		bookingRepo,
		cfg.UserServiceURL,
		cfg.ListingServiceURL,
	)
	bookingHandler := handler.NewBookingHandler(bookingSvc)

	// 5. Настраиваем роутер chi
	r := chi.NewRouter()

	// 6. Оборачиваем /bookings* в JWT-middleware
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return middleware.JWTAuthMiddleware(next, cfg.JWTSecret)
		})
		bookingHandler.RegisterRoutes(r)
	})

	// 7. Эндпоинт для проверки здоровья сервиса
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 8. Выводим, на каком порту слушаем, и запускаем HTTP-сервер
	addr := ":" + cfg.HTTPPort
	log.Printf("🚀 Booking service is running on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
