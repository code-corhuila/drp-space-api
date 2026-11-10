package app

import "errors"

var (
	ErrValidation      = errors.New("validation")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrForbidden       = errors.New("forbidden")
	ErrJWKSUnavailable = errors.New("jwks unavailable")
)
