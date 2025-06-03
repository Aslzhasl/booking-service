// internal/handler/booking.go

package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"booking-service/internal/service"
)

type BookingHandler struct {
	svc *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bookings", func(r chi.Router) {
		r.Get("/", h.listAllBookings)
		r.Post("/", h.createBooking)                     // POST   /bookings
		r.Get("/{bookingID}", h.getBookingByID)          // GET    /bookings/{bookingID}
		r.Get("/user/{userID}", h.listBookingsByUser)    // GET    /bookings/user/{userID}
		r.Get("/available", h.checkAvailabilityInterval) // GET    /bookings/available?listing_id=...&start=...&end=...
		r.Get("/availability/{listingID}", h.getDailyAvailability)
		r.Get("/available/{listingID}", h.checkAvailabilityAt) // GET    /bookings/available/{listingID}?at=...
	})
}

// createBooking обрабатывает POST /bookings
func (h *BookingHandler) createBooking(w http.ResponseWriter, r *http.Request) {
	// 1) Извлекаем заголовок Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	log.Printf("DEBUG: incoming Authorization header = %q\n", authHeader)

	// 2) Читаем тело запроса
	var reqBody struct {
		ListingID string `json:"listing_id"`
		UserID    string `json:"user_id"`
		OwnerID   string `json:"owner_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 3) Парсим даты в time.Time (RFC3339)
	start, err := time.Parse(time.RFC3339, reqBody.StartTime)
	if err != nil {
		http.Error(w, "Invalid start_time format (RFC3339 expected)", http.StatusBadRequest)
		return
	}
	end, err := time.Parse(time.RFC3339, reqBody.EndTime)
	if err != nil {
		http.Error(w, "Invalid end_time format (RFC3339 expected)", http.StatusBadRequest)
		return
	}

	// 4) Проверяем, что userID и ownerID имеют корректный UUID-формат
	if _, err := uuid.Parse(reqBody.UserID); err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(reqBody.OwnerID); err != nil {
		http.Error(w, "Invalid owner_id", http.StatusBadRequest)
		return
	}

	// 5) Готовим запрос для сервисного слоя, пробрасывая authHeader
	svcReq := &service.CreateBookingRequest{
		ListingID:  reqBody.ListingID, // treated as plain string (text)
		UserID:     reqBody.UserID,
		OwnerID:    reqBody.OwnerID,
		StartTime:  start,
		EndTime:    end,
		AuthHeader: authHeader, // "Bearer <token>"
	}

	booking, err := h.svc.CreateBooking(r.Context(), svcReq)
	if err != nil {
		http.Error(w, "Could not create booking: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

// getBookingByID обрабатывает GET /bookings/{bookingID}
func (h *BookingHandler) getBookingByID(w http.ResponseWriter, r *http.Request) {
	bookingID := chi.URLParam(r, "bookingID")
	if _, err := uuid.Parse(bookingID); err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	booking, err := h.svc.GetBookingByID(r.Context(), bookingID)
	if err != nil {
		http.Error(w, "Booking not found: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(booking)
}

// listBookingsByUser обрабатывает GET /bookings/user/{userID}
func (h *BookingHandler) listBookingsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	list, err := h.svc.ListBookingsByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Error fetching bookings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// checkAvailabilityInterval обрабатывает GET /bookings/available?listing_id=...&start=...&end=...
func (h *BookingHandler) checkAvailabilityInterval(w http.ResponseWriter, r *http.Request) {
	listingID := r.URL.Query().Get("listing_id")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if listingID == "" || startStr == "" || endStr == "" {
		http.Error(w, "Missing listing_id, start or end query parameters", http.StatusBadRequest)
		return
	}

	// Парсим даты
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start format (RFC3339 expected)", http.StatusBadRequest)
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end format (RFC3339 expected)", http.StatusBadRequest)
		return
	}

	available, err := h.svc.IsAvailableInterval(r.Context(), listingID, start, end)
	if err != nil {
		http.Error(w, "Error checking availability: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"available": available})
}

// checkAvailabilityAt обрабатывает GET /bookings/available/{listingID}?at=...
func (h *BookingHandler) checkAvailabilityAt(w http.ResponseWriter, r *http.Request) {
	listingID := chi.URLParam(r, "listingID")
	atStr := r.URL.Query().Get("at")

	if listingID == "" || atStr == "" {
		http.Error(w, "Missing listingID or 'at' query parameter", http.StatusBadRequest)
		return
	}

	timePoint, err := time.Parse(time.RFC3339, atStr)
	if err != nil {
		http.Error(w, "Invalid 'at' format (RFC3339 expected)", http.StatusBadRequest)
		return
	}

	available, err := h.svc.IsAvailableAtMoment(r.Context(), listingID, timePoint)
	if err != nil {
		http.Error(w, "Error checking availability: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"available": available})
}
func (h *BookingHandler) getDailyAvailability(w http.ResponseWriter, r *http.Request) {
	listingID := chi.URLParam(r, "listingID")
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "Missing 'date' query parameter", http.StatusBadRequest)
		return
	}

	// Проверим формат dateStr: "YYYY-MM-DD"
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		http.Error(w, "Invalid date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	hoursMap, err := h.svc.DailyAvailability(r.Context(), listingID, dateStr)
	if err != nil {
		http.Error(w, "Error getting availability: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	resp := map[string]interface{}{
		"date":  dateStr,
		"hours": hoursMap,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
func (h *BookingHandler) listAllBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.svc.ListAllBookings(r.Context())
	if err != nil {
		http.Error(w, "Error fetching all bookings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}
