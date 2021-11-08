//go:generate mockgen -source=./interface.go -destination=./mock/storage.go -package=storagemock
package storage

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gophermart/internal/app/model"
)

type UserRepository interface {
	// Create a new model.User
	Create(ctx context.Context, m *model.User) (*model.User, error)
	// ReadByNameAndPassword instance of model.User
	ReadByNameAndPassword(ctx context.Context, name string, password string) (*model.User, error)
	// Read instance of model.User
	Read(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type OrderRepository interface {
	// Create a new model.Order
	Create(ctx context.Context, m *model.Order) (*model.Order, error)
	// TxCreate a new model.Order within the tx
	TxCreate(ctx context.Context, tx *sql.Tx, m *model.Order) (*model.Order, error)
	// Read instance of model.Order
	Read(ctx context.Context, id uuid.UUID) (*model.Order, error)
	// ReadByExternalID instance of model.Order
	ReadByExternalID(ctx context.Context, externalID string) (*model.Order, error)
	// Update instance of model.Order
	Update(ctx context.Context, m *model.Order) (*model.Order, error)
	// AllByUserID returns all orders of user
	AllByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}

type TransactionRepository interface {
	// GetReplenishmentSum for user
	GetReplenishmentSum(ctx context.Context, m *model.User) (*decimal.Decimal, error)
	// GetWithdrawalSum for user
	GetWithdrawalSum(ctx context.Context, m *model.User) (*decimal.Decimal, error)
	// GetWithdrawals for user
	GetWithdrawals(ctx context.Context, m *model.User) ([]*model.Transaction, error)
	// TxCreate a new model.Transaction
	TxCreate(ctx context.Context, tx *sql.Tx, m *model.Transaction) (*model.Transaction, error)
	// Create a new model.Transaction
	Create(ctx context.Context, m *model.Transaction) (*model.Transaction, error)
}
