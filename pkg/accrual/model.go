package accrual

import "github.com/shopspring/decimal"

type GetOrderRequest struct {
	Order string `json:"order"`
}

type GetOrderResponse struct {
	Order   string              `json:"order"`
	Status  string              `json:"status"`
	Accrual decimal.NullDecimal `json:"accrual,omitempty"`
}
