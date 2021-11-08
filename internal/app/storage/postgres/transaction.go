package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ferdypruis/go-luhn"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
	"time"
)

// storage.TransactionRepository interface implementation
var _ storage.TransactionRepository = (*TransactionRepository)(nil)

type TransactionRepository struct {
	db *sql.DB
}

func (r *TransactionRepository) GetReplenishmentSum(ctx context.Context, m *model.User) (*decimal.Decimal, error) {
	const SQL = `
		SELECT sum(amount)
		FROM transactions
		WHERE type_id=$1 && user_id=$2
`
	sum := new(decimal.Decimal)

	err := r.db.QueryRowContext(ctx, SQL, model.TransactionTypeReplenishment).Scan(&sum, &m.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return sum, nil
}

func (r *TransactionRepository) GetWithdrawalSum(ctx context.Context, m *model.User) (*decimal.Decimal, error) {
	const SQL = `
		SELECT sum(amount)
		FROM transactions
		WHERE type_id=$1 && user_id=$2
`
	sum := new(decimal.Decimal)

	err := r.db.QueryRowContext(ctx, SQL, model.TransactionTypeWithdrawal, m.ID).Scan(&sum, &m.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	return sum, nil
}

func (r *TransactionRepository) GetWithdrawals(ctx context.Context, m *model.User) ([]*model.Transaction, error) {
	l := logger.Ctx(ctx).With().Str("method", "GetWithdrawalSum").Logger()

	const SQL = `
		SELECT created_at, external_order_id, amount
		FROM transactions
		WHERE type_id=$1 && user_id=$2
		ORDER BY created_at DESC
`
	rows, err := r.db.QueryContext(ctx, SQL, model.TransactionTypeWithdrawal, m.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select: %w", err)
	}

	res := make([]*model.Transaction, 0)

	for rows.Next() {
		m := &model.Transaction{}
		if err := rows.Scan(&m.CreatedAt, &m.ExternalOrderID, &m.Amount); err != nil {
			l.Debug().Err(err).Send()
			return nil, fmt.Errorf("scan: %w", err)
		}
		res = append(res, m)
	}

	return res, nil
}

func NewTransactionRepository(db *sql.DB) (*TransactionRepository, error) {
	s := &TransactionRepository{
		db: db,
	}
	return s, nil
}

// Create implementation of interface storage.TransactionRepository
func (r *TransactionRepository) Create(ctx context.Context, m *model.Transaction) (*model.Transaction, error) {
	l := logger.Ctx(ctx).With().
		Str("method", "Create").
		Str("external_order_id", m.ExternalOrderID).
		Logger()
	l.Debug().Msg("Creating transaction")

	if m.ExternalOrderID == "" || !luhn.Valid(m.ExternalOrderID) {
		return nil, apperr.ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ctx = l.WithContext(ctx)

	m.ID = uuid.New()
	m.CreatedAt = time.Now()

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		l.Error().Err(err).Msg("DB transaction begin")
		return nil, err
	}

	var balance decimal.Decimal
	const sqlLock = `SELECT balance FROM users WHERE id=$1 FOR UPDATE`
	if err := tx.QueryRowContext(ctx, sqlLock, m.UserID).Scan(&balance); err != nil {
		l.Error().Err(err).Msg("DB lock error")
		_ = tx.Rollback()
		return nil, err
	}

	if balance.LessThan(m.Amount) {
		err := apperr.ErrInsufficientFunds
		l.Error().Err(err).Msg("Insufficient funds")
		_ = tx.Rollback()
		return nil, err
	}

	const sqlTx = `INSERT INTO transactions (type_id, user_id, order_id, external_order_id, amount) VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.ExecContext(ctx, sqlTx, m.TypeID, m.UserID, m.OrderID, m.ExternalOrderID, m.Amount)
	if err != nil {
		l.Error().Err(err).Msg("TX insert failed")
		_ = tx.Rollback()
		return nil, err
	}

	const sqlUpdateBalance = `UPDATE users SET balance=balance+$1 WHERE id=$2`
	_, err = tx.ExecContext(ctx, sqlUpdateBalance, m.Amount, m.UserID)
	if err != nil {
		l.Error().Err(err).Msg("Balance update failed")
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		l.Error().Err(err).Msg("TX commit failed")
		return nil, err
	}

	dur := time.Now().Sub(m.CreatedAt)
	l.Debug().Dur("duration", dur).Msg("Done creating tx")

	return m, nil
}
