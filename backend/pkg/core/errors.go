package core

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrAlreadyExists = errors.New("already exists")
	ErrValidation    = errors.New("validation failed")
	ErrUnavailable   = errors.New("dependency unavailable")
)
