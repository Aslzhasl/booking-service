package main

import (
	"booking-service/internal/handler"
	"booking-service/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:1234@localhost:5432/booking_db?sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalln("db:", err)
	}
	repo := repository.NewBookingRepository(db)
	h := &handler.BookingHandler{
		Repo:             repo,
		UserServiceURL:   "http://localhost:8080", // адрес user-service
		DeviceServiceURL: "http://localhost:8081", // адрес device-service
	}

	http.HandleFunc("/bookings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateBooking(w, r)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	log.Println("Booking service running on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
