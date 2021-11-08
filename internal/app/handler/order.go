package handler

import (
	"errors"
	"github.com/google/uuid"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/service/syncer"
	"gophermart/internal/app/session"
	"gophermart/internal/app/storage"
	"io"
	"net/http"
	"time"
)

type OrderHandler struct {
	session session.Creator
	orders  storage.OrderRepository
	syncer  *syncer.Service
}

func NewOrderHandler(orders storage.OrderRepository, syncer *syncer.Service) *OrderHandler {
	return &OrderHandler{
		orders: orders,
		syncer: syncer,
	}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.Get(ctx, "Handler.ExternalOrderID.Create")
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
		return
	}

	m, err := h.orders.Create(ctx, &model.Order{
		ID:         uuid.New(),
		CreatedAt:  time.Now(),
		ExternalID: string(b),
		UserID:     u.ID,
	})

	if err != nil {
		if errors.Is(err, apperr.ErrInvalidInput) {
			l.Debug().Err(err).Str("order_id", string(b)).Msg("Validation error")
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

	go h.syncer.Run(h.syncer.FetchOrderDetails(m.ID))

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

	mm, err := h.orders.AllByUserID(ctx, u.ID)
	if err != nil {
		l.Debug().Err(err).Send()
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	if len(mm) == 0 {
		WriteResponse(w, struct{}{}, http.StatusNoContent)
		return
	}

	l.Debug().Msgf("response json: %s", jsonString(mm))

	WriteResponse(w, mm, http.StatusOK)
}
