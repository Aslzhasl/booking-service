package model

type Booking struct {
	ID        string `db:"id" json:"id"`
	DeviceID  string `db:"device_id" json:"device_id"`
	UserID    string `db:"user_id" json:"user_id"`
	OwnerID   string `db:"owner_id" json:"owner_id"`
	StartTime string `db:"start_time" json:"start_time"`
	EndTime   string `db:"end_time" json:"end_time"`
	Status    string `db:"status" json:"status"`
	CreatedAt string `db:"created_at" json:"created_at"`
}
