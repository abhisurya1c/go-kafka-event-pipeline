package models

import "time"

type EventType string

const (
	TypeUserCreated       EventType = "UserCreated"
	TypeOrderPlaced       EventType = "OrderPlaced"
	TypePaymentSettled    EventType = "PaymentSettled"
	TypeInventoryAdjusted EventType = "InventoryAdjusted"
)

type EventHeader struct {
	EventID   string    `json:"eventId"`
	Timestamp time.Time `json:"timestamp"`
	Type      EventType `json:"type"`
}

type UserCreated struct {
	EventHeader
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type OrderPlaced struct {
	EventHeader
	OrderID string  `json:"orderId"`
	UserID  string  `json:"userId"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
}

type PaymentSettled struct {
	EventHeader
	OrderID string  `json:"orderId"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
}

type InventoryAdjusted struct {
	EventHeader
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// DLQPayload represents the error object sent to Redis.
type DLQPayload struct {
	OriginalPayload string    `json:"original_payload"`
	Error           string    `json:"error"`
	EventID         string    `json:"eventId"`
	Timestamp       time.Time `json:"timestamp"`
}
