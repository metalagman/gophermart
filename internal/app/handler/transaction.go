package handler

import (
	"errors"
	"github.com/shopspring/decimal"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/storage"
	"net/http"
)

type TransactionHandler struct {
	transactions storage.TransactionRepository
}

func NewTransactionHandler(transactions storage.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{
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
	}

	out := struct {
		Current   decimal.Decimal
		Withdrawn decimal.Decimal
	}{
		Current:   u.Balance,
		Withdrawn: *sum,
	}

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

func (h *TransactionHandler) CreateWithdraw(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Transaction.CreateWithdraw")
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
	}

	m, err := h.transactions.Create(ctx, &model.Transaction{
		ExternalOrderID: in.ExternalOrderID,
		UserID:          u.ID,
		TypeID:          model.TransactionTypeWithdrawal,
		Amount:          in.Amount,
	})

	if err != nil {
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

		l.Debug().Err(err).Msg("Internal error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteResponse(w, m, http.StatusOK)
}
