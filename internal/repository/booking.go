package repository

import (
	"booking-service/internal/model"
	"context"
	"github.com/jmoiron/sqlx"
)

type BookingRepository struct {
	DB *sqlx.DB
}

func NewBookingRepository(db *sqlx.DB) *BookingRepository {
	return &BookingRepository{DB: db}
}

func (r *BookingRepository) CreateBooking(ctx context.Context, b *model.Booking) error {
	_, err := r.DB.NamedExecContext(ctx, `
		INSERT INTO bookings (id, device_id, user_id, owner_id, start_time, end_time, status, created_at)
		VALUES (:id, :device_id, :user_id, :owner_id, :start_time, :end_time, :status, :created_at)
	`, b)
	return err
}
func (r *BookingRepository) GetBookingByID(ctx context.Context, id string) (*model.Booking, error) {
	var booking model.Booking
	err := r.DB.GetContext(ctx, &booking, `SELECT * FROM bookings WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// Получить все брони (можно добавить фильтры при необходимости)
func (r *BookingRepository) GetAllBookings(ctx context.Context) ([]model.Booking, error) {
	var bookings []model.Booking
	err := r.DB.SelectContext(ctx, &bookings, `SELECT * FROM bookings ORDER BY created_at DESC`)
	return bookings, err
}
func (r *BookingRepository) IsDeviceBooked(ctx context.Context, deviceID, startTime, endTime string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM bookings
		WHERE device_id = $1
		  AND status = 'active'
		  AND payment_status = 'payed'
		  AND (start_time, end_time) OVERLAPS ($2, $3)
	`
	err := r.DB.GetContext(ctx, &count, query, deviceID, startTime, endTime)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Обновить бронь
func (r *BookingRepository) UpdateBooking(ctx context.Context, b *model.Booking) error {
	_, err := r.DB.NamedExecContext(ctx, `
		UPDATE bookings SET
			device_id = :device_id,
			user_id = :user_id,
			owner_id = :owner_id,
			start_time = :start_time,
			end_time = :end_time,
			status = :status,
			updated_at = NOW()
		WHERE id = :id
	`, b)
	return err
}

// Удалить бронь по ID
func (r *BookingRepository) DeleteBooking(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM bookings WHERE id = $1`, id)
	return err
}

func (r *BookingRepository) GetBookedSlotsByDeviceAndDate(ctx context.Context, deviceID, date string) ([]model.BookedSlot, error) {
	var slots []model.BookedSlot
	query := `
		SELECT start_time, end_time 
		FROM bookings
		WHERE device_id = $1
		AND (start_time::date = $2::date OR end_time::date = $2::date)
		AND status = 'active'
	`
	err := r.DB.SelectContext(ctx, &slots, query, deviceID, date)
	return slots, err
}
