package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	Password string          `json:"-"`
	Balance  decimal.Decimal `json:"-"`
}
