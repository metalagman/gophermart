package handler

import (
	"errors"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/session"
	"gophermart/internal/app/storage"
	"io"
	"net/http"
)

type OrderHandler struct {
	session session.Creator
	orders  storage.OrderRepository
}

func NewOrderHandler(orders storage.OrderRepository) *OrderHandler {
	return &OrderHandler{
		orders: orders,
	}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Order.Create")
	l.Debug().Send()

	u, err := ReadContextUser(ctx)
	if err != nil {
		l.Debug().Err(err).Msg("Unauthorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		l.Debug().Err(err).Msg("Body read failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	m, err := h.orders.Create(ctx, &model.Order{
		ExternalID: string(b),
		UserID:     u.ID,
	})

	if err != nil {
		if errors.Is(err, apperr.ErrInvalidInput) {
			l.Debug().Err(err).Msg("Validation error")
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		if errors.Is(err, apperr.ErrConflict) {
			l.Debug().Err(err).Msg("Conflict")
			eo, err := h.orders.ReadByExternalID(ctx, m.ExternalID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if eo.UserID == u.ID {
				l.Debug().Err(err).Msg("Same user")
				http.Error(w, err.Error(), http.StatusOK)
				return
			}

			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		l.Debug().Err(err).Msg("Internal error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.Order.List")
	l.Debug().Send()

	u, err := ReadContextUser(ctx)
	if err != nil {
		l.Debug().Err(err).Msg("Unauthorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	posts, err := h.orders.AllByUserID(ctx, u.ID)
	if err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	if len(posts) == 0 {
		WriteResponse(w, struct{}{}, http.StatusNoContent)
		return
	}

	WriteResponse(w, posts, http.StatusOK)
}
