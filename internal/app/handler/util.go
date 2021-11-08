package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/model"
	"io/ioutil"
	"net/http"
)

// readBody into json struct
func readBody(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return fmt.Errorf("body read: %w", err)
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	return nil
}

func jsonString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

type jsonError struct {
	Message string `json:"error"`
}

// WriteError formatted in json
func WriteError(w http.ResponseWriter, err error, statusCode int) {
	WriteResponse(w, &jsonError{Message: err.Error()}, statusCode)
}

// WriteResponse formatted in json
func WriteResponse(w http.ResponseWriter, v interface{}, statusCode int) {
	resBody, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(resBody)
}

type ValidationErrorResponse struct {
	Errors ValidationErrors `json:"errors"`
}

type ValidationErrors []ValidationError

type ValidationError struct {
	Msg   string `json:"msg"`
	Param string `json:"param"`
	Value string `json:"value"`
}

// validateData and send errors, returns true if no validation errors
func validateData(w http.ResponseWriter, v interface{}) bool {
	validate := validator.New()
	err := validate.Struct(v)
	if err != nil {
		errors := make(ValidationErrors, 0)
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, ValidationError{
				Msg:   err.Error(),
				Param: err.Field(),
				Value: fmt.Sprintf("%s", err.Value()),
			})
		}
		writeValidationErrors(w, errors)
		return false
	}

	return true
}

// writeValidationErrors formatted in json
func writeValidationErrors(w http.ResponseWriter, errors ValidationErrors) {
	WriteResponse(w, ValidationErrorResponse{errors}, http.StatusBadRequest)
}

type ContextKeyUser struct{}

func ReadContextUser(ctx context.Context) (*model.User, error) {
	v := ctx.Value(ContextKeyUser{})
	if user, ok := v.(*model.User); ok {
		return user, nil
	}

	return nil, apperr.ErrUnauthorized
}
