package entities

import "time"

type UserData struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"
)

type OrderData struct {
	ID         int       `json:"-"`
	Number     int       `json:"number"`
	Status     string    `json:"status"`
	Accural    int       `json:"accural"`
	UploadedAt time.Time `json:"uploaded_at"`
}
