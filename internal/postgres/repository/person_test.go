package repository

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic"
)

func TestClassifyWriteError_MapsNormalizedEmailUniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: emailUniqueConstraint}
	err := classifyWriteError("create person", pgErr)
	require.ErrorIs(t, err, logic.ErrEmailAlreadyExists)
}

func TestClassifyWriteError_PreservesUnrelatedDatabaseErrors(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "persons_pkey"}
	err := classifyWriteError("create person", pgErr)
	require.ErrorIs(t, err, pgErr)
	require.False(t, errors.Is(err, logic.ErrEmailAlreadyExists))
}

func TestPersonRepositoryUpdateRejectsVersionOverflowBeforeDatabaseAccess(t *testing.T) {
	person, err := app.ReconstructPerson(app.PersonSnapshot{
		ID: uuid.New(), Email: "a@example.com", Phone: "+79991234567",
		FirstName: "A", LastName: "B", Version: math.MaxInt64,
	})
	require.NoError(t, err)

	err = NewPersonRepository().Update(context.Background(), person, math.MaxInt64)

	require.ErrorContains(t, err, "version overflow")
}
