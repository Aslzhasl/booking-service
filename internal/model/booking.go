package model

import (
	"time"
)

// Booking соответствует одной записи в таблице `bookings`.
type Booking struct {
	ID        string    `db:"id" json:"id"`
	ListingID string    `db:"listing_id" json:"listing_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	OwnerID   string    `db:"owner_id" json:"owner_id"`
	StartTime time.Time `db:"start_time" json:"start_time"`
	EndTime   time.Time `db:"end_time" json:"end_time"`
	Status    string    `db:"status" json:"status"` // Новое поле: статус брони (NOT NULL)
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
