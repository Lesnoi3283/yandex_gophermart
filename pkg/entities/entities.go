package entities

import (
	"encoding/json"
	"time"
)

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

type TimeRFC3339 struct {
	Time time.Time
}

func (t *TimeRFC3339) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339))
}

func (t *TimeRFC3339) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsedTime, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsedTime
	return nil
}

type OrderData struct {
	ID         int         `json:"-"`
	UserID     int         `json:"-"`
	Number     int         `json:"number"`
	Status     string      `json:"status"`
	Accural    int         `json:"accural"`
	UploadedAt TimeRFC3339 `json:"uploaded_at"`
}

// todo: хранить во флоате или в двух интах
type BalanceData struct {
	ID        int     `json:"-"`
	UserID    int     `json:"-"`
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
