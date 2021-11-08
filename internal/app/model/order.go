package model

import (
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
