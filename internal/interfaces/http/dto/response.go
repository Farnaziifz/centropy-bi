// Package dto holds HTTP request/response shapes and the error-to-status
// mapping. Keeping these separate from application DTOs means a public API
// contract change never forces a change to internal read models, and vice
// versa.
package dto

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "centropy-affilate/pkg/errors"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// WriteError maps a domain error Kind to an HTTP status code once, here,
// so handlers never hardcode status codes for business errors.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		message = appErr.Message
		switch appErr.Kind {
		case apperrors.KindNotFound:
			status = http.StatusNotFound
		case apperrors.KindConflict:
			status = http.StatusConflict
		case apperrors.KindInvalidInput:
			status = http.StatusBadRequest
		case apperrors.KindUnauthorized:
			status = http.StatusUnauthorized
		case apperrors.KindForbidden:
			status = http.StatusForbidden
		case apperrors.KindRateLimited:
			status = http.StatusTooManyRequests
		default:
			status = http.StatusInternalServerError
			message = "internal server error"
		}
	}

	WriteJSON(w, status, ErrorResponse{Error: message})
}
