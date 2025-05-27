package main

import (
	"booking-service/internal/handler"
	"booking-service/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:1234@34.45.98.93:5432/Booking-service?sslmode=disable"
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
		switch r.Method {
		case http.MethodPost:
			h.CreateBooking(w, r)
		case http.MethodGet:
			h.GetAllBookings(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
	http.HandleFunc("/api/devices/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/booked-slots") && r.Method == http.MethodGet {
			h.GetBookedSlots(w, r)
			return
		}

		http.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) {
			// URL вида /bookings/{id}
			switch r.Method {
			case http.MethodGet:
				h.GetBookingByID(w, r)
			case http.MethodPut:
				h.UpdateBooking(w, r)
			case http.MethodPatch:
				h.PatchBooking(w, r)
			case http.MethodDelete:
				h.DeleteBooking(w, r)
			default:
				http.Error(w, "not found", http.StatusNotFound)
			}
		})

		log.Println("Booking service running on :8082")
		log.Fatal(http.ListenAndServe(":8082", nil))
	})
}
