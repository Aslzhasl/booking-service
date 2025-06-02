package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	// 1) Загружаем конфиг и подключаемся к БД
	cfg := config.LoadConfig()
	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("Connected to Postgres")

	// 2) Инициализируем репозитории, сервисы, хендлеры
	bookingRepo := repository.NewBookingRepository(db)
	bookingSvc := service.NewBookingService(
		bookingRepo,
		cfg.UserServiceURL,
		cfg.ListingServiceURL,
	)
	bookingHandler := handler.NewBookingHandler(bookingSvc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return middleware.JWTAuthMiddleware(next, cfg.JWTSecret)
		})
		bookingHandler.RegisterRoutes(r)
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 3) Определяем, на каком порту слушать:
	//    а) локально мы могли задать HTTP_PORT (например, 8082),
	//    б) в Cloud Run придёт переменная PORT=8080.
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("Booking service is listening on %s …\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
