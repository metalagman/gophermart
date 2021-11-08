package model

import (
	"github.com/google/uuid"
	"time"
)

type Transaction struct {
	ID        uuid.UUID
	CreatedAt time.Time
	TypeID    TransactionType
	OrderID   uuid.UUID
	UserID    uuid.UUID
}

type TransactionType int

const (
	TransactionTypeReplenishment TransactionType = iota + 1
	TransactionTypeWithdrawal
)
