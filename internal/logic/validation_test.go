package logic

import (
	"strings"
	"testing"

	"person/internal/app"
	"person/internal/logic/ports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateMutationReportsAllInvalidFields(t *testing.T) {
	err := ValidateMutation(uuid.Nil, 0)

	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "must be a valid UUID", validation.Fields["id"])
	require.Equal(t, "must be positive", validation.Fields["version"])
}

func TestValidateIDRejectsNilUUID(t *testing.T) {
	require.NoError(t, ValidateID(uuid.New()))

	var validation *app.ValidationError
	require.ErrorAs(t, ValidateID(uuid.Nil), &validation)
	require.Equal(t, "must be a valid UUID", validation.Fields["id"])
}

func TestValidateListQueryRejectsInvalidBounds(t *testing.T) {
	err := ValidateListQuery(ports.ListPersonsQuery{Limit: 101, Offset: -1, Search: strings.Repeat("x", MaxSearchLength+1)})

	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Fields, "limit")
	require.Contains(t, validation.Fields, "offset")
	require.Contains(t, validation.Fields, "search")
}

func TestValidateListQueryAcceptsValidBounds(t *testing.T) {
	require.NoError(t, ValidateListQuery(ports.ListPersonsQuery{Limit: 20, Offset: 0, Search: "john"}))
}
