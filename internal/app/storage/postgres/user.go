package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	pg "github.com/lib/pq"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
)

// storage.UserRepository interface implementation
var _ storage.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sql.DB
}

func (r *UserRepository) LoggerComponent() string {
	return "UserRepository"
}

func NewUserRepository(db *sql.DB) (*UserRepository, error) {
	s := &UserRepository{
		db: db,
	}

	log := logger.Global().Component(s)
	log.Debug().Msg("Creating tables")

	if err := s.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return s, nil
}

// Create implementation of interface storage.UserRepository
func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	const SQL = `
		INSERT INTO users (name, password)
		VALUES ($1, crypt($2, gen_salt('bf')))
		RETURNING id
`

	err := r.db.QueryRowContext(ctx, SQL, user.Name, user.Password).Scan(&user.ID)
	if err != nil {
		if pgErr, ok := err.(*pg.Error); ok {
			if pgerrcode.IsIntegrityConstraintViolation(string(pgErr.Code)) {
				return nil, apperr.ErrConflict
			}
		}

		return nil, fmt.Errorf("insert: %w", err)
	}

	return user, nil
}

// Get implementation of interface storage.UserRepository
func (r *UserRepository) Read(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const SQL = `
		SELECT id, name
		FROM users 
		WHERE id=$1
`
	user := &model.User{}

	err := r.db.QueryRowContext(ctx, SQL, id).Scan(&user.ID, &user.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return user, nil
}

func (r *UserRepository) ReadByNameAndPassword(ctx context.Context, name string, password string) (*model.User, error) {
	const SQL = `
		SELECT id, name
		FROM users
		WHERE name = $1 
		AND password = crypt($2, password);
`
	user := &model.User{}

	err := r.db.QueryRowContext(ctx, SQL, name, password).Scan(&user.ID, &user.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return user, nil
}

func (r *UserRepository) createTables() error {
	//_, _ = s.db.Exec(`DROP TABLE IF EXISTS "urls"`)
	const sqlCreateTable = `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
		CREATE TABLE IF NOT EXISTS "users" (
			id uuid DEFAULT uuid_generate_v4(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			name TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
`

	if _, err := r.db.Exec(sqlCreateTable); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}
