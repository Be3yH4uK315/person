package logic

import "errors"

var (
	ErrPersonNotFound      = errors.New("person not found")
	ErrConcurrencyConflict = errors.New("person version conflict")
	ErrEmailAlreadyExists  = errors.New("person email already exists")
)
