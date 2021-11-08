package session

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
	"sync"
	"time"
)

// session.Manager interface implementation
var _ Manager = (*Memory)(nil)

type (
	Memory struct {
		mu            sync.RWMutex
		issuer        string
		secretKey     []byte
		tokenLifetime time.Duration
		users         storage.UserRepository
		db            MemoryDB
	}
	MemoryDB map[string]MemorySession
)

func (svc *Memory) LoggerComponent() string {
	return "MemorySession.Memory"
}

func NewMemory(secretKey string, users storage.UserRepository, opts ...MemoryOption) *Memory {
	var (
		defaultTokenLifeTime = time.Hour
	)

	s := &Memory{
		secretKey:     []byte(secretKey),
		users:         users,
		tokenLifetime: defaultTokenLifeTime,
		db:            make(MemoryDB),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type MemorySession struct {
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    uuid.UUID `json:"user_id"`
}

// Create method of session.Creator implementation
func (svc *Memory) Create(ctx context.Context, u *model.User) (string, error) {
	l := logger.Get(ctx, svc)
	l.Debug().Str("user-id", u.ID.String()).Msg("Create")

	id := uuid.New().String()

	now := time.Now()
	exp := now.Add(svc.tokenLifetime)

	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			Id:        id,
			NotBefore: now.Unix(),
			ExpiresAt: exp.Unix(),
			Issuer:    svc.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	strToken, err := token.SignedString(svc.secretKey)
	if err != nil {
		l.Error().Err(err).Send()

		return "", fmt.Errorf("jwt encode: %w", err)
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.db[id] = MemorySession{
		UserID:    u.ID,
		StartedAt: now,
		ExpiresAt: exp,
	}

	return strToken, nil
}

// Read method of session.Reader implementation
func (svc *Memory) Read(ctx context.Context, tokenString string) (*model.User, error) {
	l := logger.Get(ctx, svc)
	l.Debug().Msg("Read request")

	c := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
		return svc.secretKey, nil
	})

	if err != nil {
		l.Debug().Err(err).Msg("ParseWithClaims failed")

		return nil, ErrInvalidToken
	}

	c, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		l.Debug().Str("token", tokenString).Msg("Invalid token")

		return nil, ErrInvalidToken
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	s, ok := svc.db[c.StandardClaims.Id]
	if !ok {
		l.Debug().Msg("MemorySession not found")

		return nil, ErrInvalidToken
	}

	if s.ExpiresAt.Before(time.Now()) {
		l.Debug().
			Str("session_id", c.StandardClaims.Id).
			Str("user_id", s.UserID.String()).
			Msg("Session expired")
		delete(svc.db, c.StandardClaims.Id)

		return nil, ErrInvalidToken
	}

	u, err := svc.users.Read(ctx, s.UserID)
	if err != nil {
		l.Debug().Err(err).Send()

		return nil, ErrInvalidToken
	}

	l.Debug().Msgf("user: %#v", *u)

	return u, nil
}
