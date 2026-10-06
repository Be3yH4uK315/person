package logic

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"person/internal/app"
	"person/internal/logic/ports"

	"github.com/google/uuid"
)

const (
	DefaultLimit    = 20
	MaxLimit        = 100
	MaxOffset       = 1_000_000
	MaxSearchLength = 128
)

func ValidateID(id uuid.UUID) error {
	details := make(map[string]string)
	addIDValidation(details, id)
	if len(details) > 0 {
		return app.NewValidationError(details)
	}
	return nil
}

func addIDValidation(details map[string]string, id uuid.UUID) {
	if id == uuid.Nil {
		details["id"] = "must be a valid UUID"
	}
}

func ValidateMutation(id uuid.UUID, version int64) error {
	details := make(map[string]string)
	addIDValidation(details, id)
	if version < 1 {
		details["version"] = "must be positive"
	}
	if len(details) > 0 {
		return app.NewValidationError(details)
	}
	return nil
}

func ValidateListQuery(query ports.ListPersonsQuery) error {
	details := make(map[string]string)
	if query.Limit < 1 || query.Limit > MaxLimit {
		details["limit"] = fmt.Sprintf("must be between 1 and %d", MaxLimit)
	}
	if query.Offset < 0 || query.Offset > MaxOffset {
		details["offset"] = fmt.Sprintf("must be between 0 and %d", MaxOffset)
	}
	if utf8.RuneCountInString(strings.TrimSpace(query.Search)) > MaxSearchLength {
		details["search"] = fmt.Sprintf("must not exceed %d characters", MaxSearchLength)
	}
	if len(details) > 0 {
		return app.NewValidationError(details)
	}
	return nil
}
