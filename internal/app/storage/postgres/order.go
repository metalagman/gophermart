package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ferdypruis/go-luhn"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	pg "github.com/lib/pq"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
)

// storage.OrderRepository interface implementation
var _ storage.OrderRepository = (*OrderRepository)(nil)

type OrderRepository struct {
	db *sql.DB
}

func (r *OrderRepository) LoggerComponent() string {
	return "OrderRepository"
}

func NewOrderRepository(db *sql.DB) (*OrderRepository, error) {
	s := &OrderRepository{
		db: db,
	}
	return s, nil
}

// Create implementation of interface storage.OrderRepository
func (r *OrderRepository) Create(ctx context.Context, m *model.Order) (*model.Order, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("tx begin: %w", err)
	}

	res, err := r.TxCreate(ctx, tx, m)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("tx create: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return res, nil
}

// TxCreate implementation of interface storage.OrderRepository
func (r *OrderRepository) TxCreate(ctx context.Context, tx *sql.Tx, m *model.Order) (*model.Order, error) {
	if m.ExternalID == "" || !luhn.Valid(m.ExternalID) {
		return nil, apperr.ErrInvalidInput
	}

	const SQL = `
		INSERT INTO orders (external_id, user_id)
		VALUES ($1, $2)
		RETURNING id
`

	err := tx.QueryRowContext(ctx, SQL, m.ExternalID, m.UserID).Scan(&m.ID)
	if err != nil {
		if pgErr, ok := err.(*pg.Error); ok {
			if pgerrcode.IsIntegrityConstraintViolation(string(pgErr.Code)) {
				em, err := r.ReadByExternalID(ctx, m.ExternalID)
				if err != nil {
					return nil, err
				}
				if em.UserID == m.UserID {
					return nil, apperr.ErrSoftConflict
				}
				return nil, apperr.ErrConflict
			}
		}

		return nil, fmt.Errorf("insert: %w", err)
	}

	return m, nil
}

// Read implementation of interface storage.OrderRepository
func (r *OrderRepository) Read(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	const SQL = `
		SELECT id, external_id, created_at, user_id, status, accrual
		FROM orders 
		WHERE id=$1
`
	m := &model.Order{}

	err := r.db.QueryRowContext(ctx, SQL, id).Scan(&m.ID, &m.ExternalID, m.CreatedAt, m.UserID, m.Status, m.Accrual)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return m, nil
}

// ReadByExternalID implementation of interface storage.OrderRepository
func (r *OrderRepository) ReadByExternalID(ctx context.Context, externalID string) (*model.Order, error) {
	const SQL = `
		SELECT id, external_id, created_at, user_id, status, accrual
		FROM orders 
		WHERE external_id=$1
`
	m := &model.Order{}

	err := r.db.QueryRowContext(ctx, SQL, externalID).Scan(&m.ID, &m.ExternalID, m.CreatedAt, m.UserID, m.Status, m.Accrual)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return m, nil
}

// Update implementation of interface storage.OrderRepository
func (r *OrderRepository) Update(ctx context.Context, m *model.Order) (*model.Order, error) {
	const SQL = `
		UPDATE orders 
		SET status=$1,accrual=$2
		WHERE id=$3
`

	_, err := r.db.ExecContext(ctx, SQL, m.Status, m.Accrual, m.ID)
	if err != nil {
		if pgErr, ok := err.(*pg.Error); ok {
			if pgerrcode.IsIntegrityConstraintViolation(string(pgErr.Code)) {
				return nil, apperr.ErrConflict
			}
		}

		return nil, fmt.Errorf("update: %w", err)
	}

	return m, nil
}

func (r *OrderRepository) AllByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	l := logger.Ctx(ctx).With().Str("method", "AllByUserID").Logger()

	const SQL = `
		SELECT id, external_id, created_at, user_id, status, accrual
		FROM orders 
		WHERE user_id=$1
		ORDER BY created_at
`
	rows, err := r.db.QueryContext(ctx, SQL, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Debug().Err(err).Send()
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := make([]*model.Order, 0)

	for rows.Next() {
		m := &model.Order{}
		if err := rows.Scan(&m.ID, &m.ExternalID, &m.CreatedAt, &m.UserID, &m.Status, &m.Accrual); err != nil {
			l.Debug().Err(err).Send()
			return nil, fmt.Errorf("scan: %w", err)
		}
		l.Debug().Msgf("order: %#v", *m)
		res = append(res, m)
	}

	return res, nil
}
