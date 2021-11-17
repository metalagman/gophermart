package handler

import (
	"errors"
	"github.com/rs/zerolog/hlog"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/internal/app/session"
	"gophermart/internal/app/storage"
	"net/http"
)

type UserHandler struct {
	session session.Creator
	users   storage.UserRepository
}

func NewUserHandler(users storage.UserRepository, sm session.Creator) *UserHandler {
	return &UserHandler{
		session: sm,
		users:   users,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := logger.Get(r.Context(), "Handler.User.Register")

	in := struct {
		Username string `json:"login" validate:"required,min=1,max=32,alphanum"`
		Password string `json:"password" validate:"required,min=8,max=72"`
	}{}

	if err := readBody(r, &in); err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	if !validateData(w, in) {
		return
	}

	u := &model.User{
		Name:     in.Username,
		Password: in.Password,
	}

	u, err := h.users.Create(r.Context(), u)

	if err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			log.Debug().Err(err).Send()
			WriteError(w, err, http.StatusConflict)
			return
		}
		if errors.Is(err, apperr.ErrInvalidInput) {
			log.Debug().Err(err).Send()
			WriteError(w, err, http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Send()
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	token, err := h.session.Create(r.Context(), u)
	if err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	out := struct {
		Token string `json:"token"`
	}{token}

	w.Header().Add("Authorization", "Bearer "+token)

	WriteResponse(w, out, http.StatusOK)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	hlog.FromRequest(r).Debug().Msg("Handler.User.Login")

	in := struct {
		Username string `json:"login" validate:"required,min=1,max=32,alphanum"`
		Password string `json:"password" validate:"required,min=8,max=72"`
	}{}

	if err := readBody(r, &in); err != nil {
		WriteError(w, err, http.StatusBadRequest)
		return
	}

	if !validateData(w, in) {
		return
	}

	u, err := h.users.ReadByNameAndPassword(r.Context(), in.Username, in.Password)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			WriteError(w, err, http.StatusUnauthorized)
			return
		}
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	token, err := h.session.Create(r.Context(), u)
	if err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	out := struct {
		Token string `json:"token"`
	}{token}

	w.Header().Add("Authorization", "Bearer "+token)

	WriteResponse(w, out, http.StatusOK)
}
