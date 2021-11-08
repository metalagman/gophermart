package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type Transaction struct {
	ID              uuid.UUID       `json:"-"`
	CreatedAt       time.Time       `json:"processed_at"`
	TypeID          TransactionType `json:"-"`
	ExternalOrderID string          `json:"order"`
	OrderID         uuid.UUID       `json:"-"`
	UserID          uuid.UUID       `json:"-"`
	Amount          decimal.Decimal `json:"sum"`
}

type TransactionType int

const (
	TransactionTypeReplenishment TransactionType = iota + 1
	TransactionTypeWithdrawal
)
