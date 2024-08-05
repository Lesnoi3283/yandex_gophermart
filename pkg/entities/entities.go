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

	OrderTimeFormat = time.RFC3339
)

type TimeRFC3339 struct {
	Time time.Time
}

func (t *TimeRFC3339) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(OrderTimeFormat))
}

func (t *TimeRFC3339) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsedTime, err := time.Parse(OrderTimeFormat, s)
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
	Accural    float64     `json:"accural"`
	UploadedAt TimeRFC3339 `json:"uploaded_at"`
}

// todo: хранить во флоате или в двух интах
type BalanceData struct {
	ID        int     `json:"-"`
	UserID    int     `json:"-"`
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawalData struct {
	OrderID     int         `json:"order"`
	Sum         float64     `json:"sum"`
	ProcessedAt TimeRFC3339 `json:"processed_at"`
}
