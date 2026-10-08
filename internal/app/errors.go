package app

import "errors"

var (
	ErrValidation      = errors.New("validation")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFound        = errors.New("not found")
	ErrJWKSUnavailable = errors.New("jwks unavailable")
)
