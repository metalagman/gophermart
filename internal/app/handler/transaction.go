package handler

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
	"net/http"
	"time"
)

type TransactionHandler struct {
	db           *sql.DB
	orders       storage.OrderRepository
	transactions storage.TransactionRepository
}

func NewTransactionHandler(db *sql.DB, transactions storage.TransactionRepository, orders storage.OrderRepository) *TransactionHandler {
	return &TransactionHandler{
		db:           db,
		orders:       orders,
		transactions: transactions,
	}
}

func (h *TransactionHandler) Balance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Transaction.Balance")
	l.Debug().Send()

	u, err := ReadContextUser(ctx)
	if err != nil {
		l.Debug().Err(err).Msg("Unauthorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	sum, err := h.transactions.GetWithdrawalSum(ctx, u)
	if err != nil {
		l.Debug().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := struct {
		Current   decimal.Decimal `json:"current"`
		Withdrawn decimal.Decimal `json:"withdrawn"`
	}{
		Current:   u.Balance,
		Withdrawn: *sum,
	}

	l.Debug().Msgf("sending balance %s", jsonString(out))
	WriteResponse(w, out, http.StatusOK)
}

func (h *TransactionHandler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Transaction.ListWithdrawals")
	l.Debug().Send()

	u, err := ReadContextUser(ctx)
	if err != nil {
		l.Debug().Err(err).Msg("Unauthorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	mm, err := h.transactions.GetWithdrawals(ctx, u)
	if err != nil {
		l.Debug().Err(err).Send()
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	WriteResponse(w, mm, http.StatusOK)
}

func (h *TransactionHandler) CreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Transaction.CreateWithdrawal")
	l.Debug().Send()

	u, err := ReadContextUser(ctx)
	if err != nil {
		l.Debug().Err(err).Msg("Unauthorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	in := &struct {
		ExternalOrderID string          `json:"order"`
		Amount          decimal.Decimal `json:"sum"`
	}{}

	if err := readBody(r, in); err != nil {
		l.Debug().Err(err).Msg("Body read failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	om, err := h.orders.TxCreate(ctx, tx, &model.Order{
		ID:         uuid.New(),
		CreatedAt:  time.Now(),
		ExternalID: in.ExternalOrderID,
		UserID:     u.ID,
	})

	if err != nil {
		_ = tx.Rollback()

		if errors.Is(err, apperr.ErrInvalidInput) {
			l.Debug().Err(err).Str("order_id", in.ExternalOrderID).Msg("Validation error")
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		l.Debug().Err(err).Msg("Internal error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m, err := h.transactions.TxCreate(ctx, tx, &model.Transaction{
		OrderID:         om.ID,
		ExternalOrderID: om.ExternalID,
		UserID:          om.UserID,
		TypeID:          model.TransactionTypeWithdrawal,
		Amount:          in.Amount,
	})

	if err != nil {
		_ = tx.Rollback()

		if errors.Is(err, apperr.ErrInsufficientFunds) {
			l.Debug().Err(err).Msg("Insufficient funds")
			http.Error(w, err.Error(), http.StatusPaymentRequired)
			return
		}

		if errors.Is(err, apperr.ErrSoftConflict) {
			l.Debug().Err(err).Msg("Soft conflict")
			http.Error(w, err.Error(), http.StatusOK)
			return
		}

		if errors.Is(err, apperr.ErrConflict) {
			l.Debug().Err(err).Msg("Hard conflict")
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		if errors.Is(err, apperr.ErrInvalidInput) {
			l.Debug().Err(err).Str("order_id", in.ExternalOrderID).Msg("Validation error")
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		l.Error().Err(err).Msg("Internal error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		l.Error().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteResponse(w, m, http.StatusOK)
}
