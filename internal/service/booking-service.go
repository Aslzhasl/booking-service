package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"booking-service/internal/model"
	"booking-service/internal/repository"
)

type CreateBookingRequest struct {
	ListingID  string    `json:"listing_id"`
	UserID     string    `json:"user_id"`
	OwnerID    string    `json:"owner_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	AuthHeader string    // Bearer <token>
}
type BookingService struct {
	repo              *repository.BookingRepository
	userServiceURL    string
	listingServiceURL string
	httpClient        *http.Client
}

func NewBookingService(
	repo *repository.BookingRepository,
	userSvcURL, listingSvcURL string,
) *BookingService {
	return &BookingService{
		repo:              repo,
		userServiceURL:    userSvcURL,
		listingServiceURL: listingSvcURL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *BookingService) CreateBooking(ctx context.Context, req *CreateBookingRequest) (*model.Booking, error) {
	// 1) Проверяем, что end_time > start_time
	if !req.EndTime.After(req.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	// 2) Проверка через User Service (убедиться, что userID и ownerID существуют)
	if err := s.checkUserExists(req.UserID, req.AuthHeader); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}
	if err := s.checkUserExists(req.OwnerID, req.AuthHeader); err != nil {
		return nil, fmt.Errorf("owner validation failed: %w", err)
	}

	// 3) Проверка через Listing Service (убедиться, что listingID существует)
	if err := s.checkListingExists(req.ListingID, req.AuthHeader); err != nil {
		return nil, fmt.Errorf("listing validation failed: %w", err)
	}

	// 4) Проверяем, нет ли пересечений в таблице bookings
	overlap, err := s.repo.HasOverlap(ctx, req.ListingID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("error checking overlap: %w", err)
	}
	if overlap {
		return nil, errors.New("listing is already booked for the given time range")
	}

	// 5) Формируем объект Booking и сразу задаём Status = "PENDING"
	booking := &model.Booking{
		ListingID: req.ListingID,
		UserID:    req.UserID,
		OwnerID:   req.OwnerID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    "PENDING", // <-- Здесь задаём начальный статус
	}

	// 6) Вставляем запись в БД
	if err := s.repo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	return booking, nil
}

func (s *BookingService) GetBookingByID(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BookingService) ListBookingsByUser(ctx context.Context, userID string) ([]model.Booking, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *BookingService) IsAvailableInterval(ctx context.Context, listingID string, start, end time.Time) (bool, error) {
	// Проверяем существование listing через Listing Service
	if err := s.checkListingExists(listingID, ""); err != nil {
		return false, fmt.Errorf("listing validation failed: %w", err)
	}

	overlap, err := s.repo.HasOverlap(ctx, listingID, start, end)
	if err != nil {
		return false, fmt.Errorf("error checking overlap: %w", err)
	}
	return !overlap, nil
}

func (s *BookingService) IsAvailableAtMoment(ctx context.Context, listingID string, timePoint time.Time) (bool, error) {
	if err := s.checkListingExists(listingID, ""); err != nil {
		return false, fmt.Errorf("listing validation failed: %w", err)
	}
	return s.repo.IsAvailableAt(ctx, listingID, timePoint)
}

// checkUserExists запрашивает GET /api/users/{userID}
func (s *BookingService) checkUserExists(userID, authHeader string) error {
	url := fmt.Sprintf("%s/api/users/%s", s.userServiceURL, userID)
	log.Printf("DEBUG: checkUserExists: URL=%s, Authorization=%q\n", url, authHeader)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
		log.Printf("DEBUG: checkUserExists: outgoing request Header Authorization=%q\n", req.Header.Get("Authorization"))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: checkUserExists: response status = %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user-service returned status %d", resp.StatusCode)
	}
	return nil
}

// checkListingExists запрашивает GET /api/listings/{listingID}
func (s *BookingService) checkListingExists(listingID, authHeader string) error {
	url := fmt.Sprintf("%s/api/listings/%s", s.listingServiceURL, listingID)
	log.Printf("DEBUG: checkListingExists: URL=%s, Authorization=%q\n", url, authHeader)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
		log.Printf("DEBUG: checkListingExists: outgoing request Header Authorization=%q\n", req.Header.Get("Authorization"))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: checkListingExists: response status = %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("listing-service returned status %d", resp.StatusCode)
	}
	return nil
}
func (s *BookingService) DailyAvailability(ctx context.Context, listingID, dateStr string) (map[string]bool, error) {
	// 1. Парсим dateStr как дата без времени (формат “2006-01-02”).
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}
	// Явно ставим UTC, чтобы не было смещения:
	date = date.UTC()

	// 2. Получаем все брони для этого listingID, которые хоть на секунду пересекаются с этим днём.
	bookings, err := s.repo.ListByListingAndDate(ctx, listingID, date)
	if err != nil {
		return nil, fmt.Errorf("DailyAvailability: %w", err)
	}

	// 3. Заранее заводим «часы» с 09 до 21 (или другие, по вашей логике).
	hourMap := map[string]bool{}
	for h := 9; h <= 21; h++ {
		// форматируем: "09:00", "10:00" и т.д.
		key := fmt.Sprintf("%02d:00", h)
		hourMap[key] = true // по умолчанию все слоты считаем свободными
	}

	// 4. Теперь обходя все найденные брони, ставим занятые те часы, которые пересекаются
	//    с каждым бронированием:
	//
	for _, b := range bookings {
		// b.StartTime и b.EndTime — в UTC, потому что мы всё хранить в UTC.
		// Находим час начала: например, 2025-02-22T11:30 UTC → заносим 11:00, 12:00 и т.д.
		startHour := b.StartTime.Hour()
		endHour := b.EndTime.Hour()
		//
		// Если бронирование началось, скажем, 11:30, мы считаем, что слот "11:00" занят
		// (его полностью не поместить). Если закончится 14:00, слот "14:00" рехзнести?
		// Часто считается, что интервал [start, end) значит, что конец не включает конечный час,
		// т. е. если endTime == 14:00, то слот "14:00" считается свободным.
		//
		// Поэтому будем заносить занятыми слоты с startHour до (endHour-1) включительно:
		for h := startHour; h < endHour; h++ {
			key := fmt.Sprintf("%02d:00", h)
			// если ключ есть в карте современных слотов (9..21), то помечаем false
			if _, ok := hourMap[key]; ok {
				hourMap[key] = false
			}
		}
	}

	// 5. Вернём итоговую карту
	return hourMap, nil
}
func (s *BookingService) ListAllBookings(ctx context.Context) ([]model.Booking, error) {
	bookings, err := s.repo.ListAllBookings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching all bookings: %w", err)
	}
	return bookings, nil
}
