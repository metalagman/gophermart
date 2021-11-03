//go:generate mockgen -source=./interface.go -destination=./mock/storage.go -package=storagemock
package storage

import (
	"context"
	"github.com/google/uuid"
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
