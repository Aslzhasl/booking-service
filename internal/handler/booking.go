package handler

import (
	"booking-service/internal/model"
	"booking-service/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
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
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
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

	// JWT из оригинального запроса
	jwtToken := r.Header.Get("Authorization")
	if jwtToken != "" && len(jwtToken) > 7 && jwtToken[:7] == "Bearer " {
		jwtToken = jwtToken[7:] // Убираем Bearer
	}

	deviceURL := fmt.Sprintf("%s/api/devices/%s", h.DeviceServiceURL, req.DeviceID)
	userURL := fmt.Sprintf("%s/api/users/%s", h.UserServiceURL, req.UserID)
	if !existsInService(deviceURL, jwtToken) {
		http.Error(w, "device not found", http.StatusBadRequest)
		return
	}
	if !existsInService(userURL, jwtToken) {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	booking := model.Booking{
		ID:        uuid.New().String(),
		DeviceID:  req.DeviceID,
		UserID:    req.UserID,
		OwnerID:   req.OwnerID,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
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
func existsInService(url string, jwtToken string) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	if jwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+jwtToken)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
