package model

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type Order struct {
	ID         uuid.UUID           `json:"-"`
	ExternalID string              `json:"number"`
	CreatedAt  time.Time           `json:"uploaded_at"`
	UserID     uuid.UUID           `json:"-"`
	Status     string              `json:"status"`
	Accrual    decimal.NullDecimal `json:"accrual,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
func (d Order) MarshalJSON() ([]byte, error) {
	o := struct {
		ExternalID string    `json:"number"`
		CreatedAt  time.Time `json:"uploaded_at"`
		Status     string    `json:"status"`
		Accrual    float64   `json:"accrual,omitempty"`
	}{
		ExternalID: d.ExternalID,
		CreatedAt:  d.CreatedAt,
		Status:     d.Status,
		Accrual:    d.Accrual.Decimal.InexactFloat64(),
	}

	return json.Marshal(o)
}
