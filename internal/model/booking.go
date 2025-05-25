package model

type Booking struct {
	ID        string `db:"id" json:"id"`
	DeviceID  string `db:"device_id" json:"device_id"`
	UserID    string `db:"user_id" json:"user_id"`
	OwnerID   string `db:"owner_id" json:"owner_id"`
	StartDate string `db:"start_date" json:"start_date"`
	EndDate   string `db:"end_date" json:"end_date"`
	Status    string `db:"status" json:"status"`
	CreatedAt string `db:"created_at" json:"created_at"`
}
