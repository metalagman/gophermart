//go:generate mockgen -source=./interface.go -destination=./mock/session.go -package=sessionmock
package session

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	"gophermart/internal/app/model"
)

var ErrInvalidToken = errors.New("invalid token")

type Manager interface {
	Creator
	Reader
}

type Creator interface {
	// Create session for provided user
	Create(ctx context.Context, user *model.User) (string, error)
}

type Reader interface {
	// Read provided session token, return user on success
	Read(ctx context.Context, token string) (*model.User, error)
}

type Claims struct {
	jwt.StandardClaims
}
