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
		INSERT INTO bookings (id, device_id, user_id, owner_id, start_date, end_date, status, created_at)
		VALUES (:id, :device_id, :user_id, :owner_id, :start_date, :end_date, :status, :created_at)
	`, b)
	return err
}
