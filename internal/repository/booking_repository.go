package repository

import (
	"context"
	"fmt"
	"time"

	"booking-service/internal/model"
	"github.com/jmoiron/sqlx"
)

type BookingRepository struct {
	db *sqlx.DB
}

func NewBookingRepository(db *sqlx.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create вставляет новую запись в таблицу bookings и возвращает сгенерированный ID, created_at, updated_at.
func (r *BookingRepository) Create(ctx context.Context, b *model.Booking) error {
	query := `
		INSERT INTO bookings
			(listing_id, user_id, owner_id, start_time, end_time, status)
		VALUES
			($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	// Выполняем INSERT и читаем обратно поля ID/created_at/updated_at
	err := r.db.QueryRowxContext(
		ctx,
		query,
		b.ListingID,
		b.UserID,
		b.OwnerID,
		b.StartTime,
		b.EndTime,
		b.Status, // Передаём статус в базу
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		return fmt.Errorf("BookingRepository.Create: %w", err)
	}
	return nil
}

// HasOverlap проверяет, существуют ли записи, пересекающиеся с [start, end) для данного listingID.
func (r *BookingRepository) HasOverlap(ctx context.Context, listingID string, start, end time.Time) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM bookings
			WHERE listing_id = $1
			  AND tstzrange(start_time, end_time, '[]') && tstzrange($2, $3, '[]')
		)
	`
	if err := r.db.GetContext(ctx, &exists, query, listingID, start, end); err != nil {
		return false, fmt.Errorf("BookingRepository.HasOverlap: %w", err)
	}
	return exists, nil
}

// GetByID возвращает одну бронь по её ID.
func (r *BookingRepository) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	var b model.Booking
	query := "SELECT * FROM bookings WHERE id = $1"
	if err := r.db.GetContext(ctx, &b, query, id); err != nil {
		return nil, fmt.Errorf("BookingRepository.GetByID: %w", err)
	}
	return &b, nil
}

// ListByUserID возвращает все брони, сделанные пользователем с userID.
func (r *BookingRepository) ListByUserID(ctx context.Context, userID string) ([]model.Booking, error) {
	var list []model.Booking
	query := "SELECT * FROM bookings WHERE user_id = $1 ORDER BY start_time DESC"
	if err := r.db.SelectContext(ctx, &list, query, userID); err != nil {
		return nil, fmt.Errorf("BookingRepository.ListByUserID: %w", err)
	}
	return list, nil
}

// IsAvailableAt проверяет, свободен ли listingID в момент timePoint.
func (r *BookingRepository) IsAvailableAt(ctx context.Context, listingID string, timePoint time.Time) (bool, error) {
	var overlap bool
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM bookings 
			WHERE listing_id = $1 
			  AND $2 BETWEEN start_time AND end_time
		)
	`
	if err := r.db.GetContext(ctx, &overlap, query, listingID, timePoint); err != nil {
		return false, fmt.Errorf("BookingRepository.IsAvailableAt: %w", err)
	}
	return !overlap, nil
}
func (r *BookingRepository) ListByListingAndDate(ctx context.Context, listingID string, date time.Time) ([]model.Booking, error) {
	// «date» мы будем передавать так, что date = 2025-02-22 00:00:00 UTC.
	// Тогда следующий день будет datePlus = 2025-02-23 00:00:00 UTC.
	datePlus := date.Add(24 * time.Hour)

	// Условие пересечения (start_time < datePlus) AND (end_time > date)
	// Значит, часть брони лежит хоть одним часом на этом дне.
	query := `
		SELECT * 
		FROM bookings
		WHERE listing_id = $1
		  AND start_time < $2
		  AND end_time   > $3
		ORDER BY start_time
	`
	var list []model.Booking
	if err := r.db.SelectContext(ctx, &list, query, listingID, datePlus, date); err != nil {
		return nil, fmt.Errorf("BookingRepository.ListByListingAndDate: %w", err)
	}
	return list, nil
}

func (r *BookingRepository) ListAllBookings(ctx context.Context) ([]model.Booking, error) {
	var bookings []model.Booking
	query := "SELECT * FROM bookings ORDER BY created_at DESC"
	if err := r.db.SelectContext(ctx, &bookings, query); err != nil {
		return nil, fmt.Errorf("BookingRepository.ListAllBookings: %w", err)
	}
	return bookings, nil
}
