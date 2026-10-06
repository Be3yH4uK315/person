package app

import "errors"

var (
	ErrCorruptedPerson    = errors.New("corrupted person")
	ErrInvalidPersonState = errors.New("invalid person state")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

func NewValidationError(fields map[string]string) error {
	return &ValidationError{Fields: fields}
}
