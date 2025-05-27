package handler

import (
	"booking-service/internal/model"
	"booking-service/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

type BookingHandler struct {
	Repo             *repository.BookingRepository
	UserServiceURL   string
	DeviceServiceURL string
}

type CreateBookingRequest struct {
	DeviceID  string `json:"device_id"`
	UserID    string `json:"user_id"`
	OwnerID   string `json:"owner_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// Проверка, что сущность существует (200 OK)
//
//	func existsInService(url string) bool {
//		resp, err := http.Get(url)
//		if err != nil {
//			return false
//		}
//		defer resp.Body.Close()
//		return resp.StatusCode == http.StatusOK
//	}
func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	if req.UserID == req.OwnerID {
		http.Error(w, "user_id and owner_id must be different", http.StatusBadRequest)
		return
	}
	booked, err := h.Repo.IsDeviceBooked(r.Context(), req.DeviceID, req.StartTime, req.EndTime)
	if err != nil {
		http.Error(w, "error checking availability: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if booked {
		http.Error(w, "device is already booked for these dates", http.StatusConflict)
		return
	}

	// JWT из оригинального запроса
	jwtToken := r.Header.Get("Authorization")
	if jwtToken != "" && len(jwtToken) > 7 && jwtToken[:7] == "Bearer " {
		jwtToken = jwtToken[7:] // Убираем Bearer
	}

	deviceURL := fmt.Sprintf("%s/api/devices/%s", h.DeviceServiceURL, req.DeviceID)
	userURL := fmt.Sprintf("%s/api/users/%s", h.UserServiceURL, req.UserID)

	deviceExists, deviceStatus := existsInService(deviceURL, jwtToken)
	if !deviceExists {
		if deviceStatus == http.StatusUnauthorized {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		if deviceStatus == http.StatusNotFound {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		http.Error(w, "error checking device", http.StatusInternalServerError)
		return
	}

	userExists, userStatus := existsInService(userURL, jwtToken)
	if !userExists {
		if userStatus == http.StatusUnauthorized {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		if userStatus == http.StatusNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "error checking user", http.StatusInternalServerError)
		return
	}

	booking := model.Booking{
		ID:        uuid.New().String(),
		DeviceID:  req.DeviceID,
		UserID:    req.UserID,
		OwnerID:   req.OwnerID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := h.Repo.CreateBooking(r.Context(), &booking); err != nil {
		// Показываем ошибку и в http ответе, и в консоль
		http.Error(w, "failed to create booking: "+err.Error(), http.StatusInternalServerError)
		fmt.Println("FAILED TO CREATE BOOKING:", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}
func (h *BookingHandler) GetAllBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.Repo.GetAllBookings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(bookings)
}

// --- GET /bookings/{id}
func (h *BookingHandler) GetBookingByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/bookings/")
	booking, err := h.Repo.GetBookingByID(r.Context(), id)
	if err != nil {
		http.Error(w, "booking not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(booking)
}

// --- PUT /bookings/{id}
func (h *BookingHandler) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/bookings/")
	var b model.Booking
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	b.ID = id
	if err := h.Repo.UpdateBooking(r.Context(), &b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(b)
}

// --- PATCH /bookings/{id} (частичное обновление, например, только статус)
type PatchBookingRequest struct {
	Status string `json:"status"`
}

func (h *BookingHandler) PatchBooking(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/bookings/")
	var req PatchBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	booking, err := h.Repo.GetBookingByID(r.Context(), id)
	if err != nil {
		http.Error(w, "booking not found", http.StatusNotFound)
		return
	}
	booking.Status = req.Status
	if err := h.Repo.UpdateBooking(r.Context(), booking); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(booking)
}

// --- DELETE /bookings/{id}
func (h *BookingHandler) DeleteBooking(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/bookings/")
	if err := h.Repo.DeleteBooking(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func existsInService(url string, jwtToken string) (bool, int) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0
	}
	if jwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+jwtToken)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, resp.StatusCode
}

func extractToken(header string) string {
	if strings.HasPrefix(header, "Bearer ") {
		return header[7:]
	}
	return header
}

func (h *BookingHandler) GetBookedSlots(w http.ResponseWriter, r *http.Request) {
	// Получаем device_id из URL
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	deviceID = strings.Split(deviceID, "/")[0]

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "date query param required", http.StatusBadRequest)
		return
	}

	slots, err := h.Repo.GetBookedSlotsByDeviceAndDate(r.Context(), deviceID, date)
	if err != nil {
		http.Error(w, "failed to fetch booked slots: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"device_id":    deviceID,
		"date":         date,
		"booked_slots": slots,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
