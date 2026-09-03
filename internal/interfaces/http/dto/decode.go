package dto

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"

	apperrors "centropy-affilate/pkg/errors"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// DecodeAndValidate reads a JSON body into T and runs struct validation
// tags, returning a ready-to-use apperrors.InvalidInput on either failure
// so handlers can pass it straight to WriteError.
func DecodeAndValidate[T any](r *http.Request) (T, error) {
	var body T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return body, apperrors.InvalidInput("invalid request body")
	}
	if err := validate.Struct(body); err != nil {
		return body, apperrors.InvalidInput(err.Error())
	}
	return body, nil
}
