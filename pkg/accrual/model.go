package accrual

import "github.com/shopspring/decimal"

type GetOrderRequest struct {
	ExternalOrderID string `json:"order"`
}

type GetOrderResponse struct {
	ExternalOrderID string              `json:"order"`
	Status          string              `json:"status"`
	Accrual         decimal.NullDecimal `json:"accrual,omitempty"`
}
