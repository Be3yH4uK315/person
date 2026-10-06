package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic"
)

func validCreateCommand() CreatePersonCommand {
	return CreatePersonCommand{
		Email:     " John@example.com ",
		Phone:     "+79991234567",
		FirstName: "John",
		LastName:  "Doe",
	}
}

func TestCreatePersonHandler_CreatesPerson(t *testing.T) {
	repository := &repositoryMock{}
	handler := NewCreatePersonHandler(repository, uowMock{})

	id, err := handler.Handle(context.Background(), validCreateCommand())

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
	require.Equal(t, id, repository.createdPerson.ID())
	require.True(t, repository.created)
	require.Equal(t, "john@example.com", repository.createdPerson.Email())
}

func TestCreatePersonHandler_PropagatesRepositoryErrorAndRollsBack(t *testing.T) {
	wantErr := errors.New("repository failure")
	var committed, rolledBack bool
	handler := NewCreatePersonHandler(
		&repositoryMock{createErr: wantErr},
		uowMock{committed: &committed, rolledBack: &rolledBack},
	)

	id, err := handler.Handle(context.Background(), validCreateCommand())

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, uuid.Nil, id)
	require.False(t, committed)
	require.True(t, rolledBack)
}

func TestCreatePersonHandler_RejectsInvalidInputWithoutTouchingRepository(t *testing.T) {
	repository := &repositoryMock{}
	handler := NewCreatePersonHandler(repository, uowMock{})

	_, err := handler.Handle(context.Background(), CreatePersonCommand{})

	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.False(t, repository.created)
}

func TestCreatePersonHandler_PropagatesDuplicateEmail(t *testing.T) {
	handler := NewCreatePersonHandler(&repositoryMock{createErr: logic.ErrEmailAlreadyExists}, uowMock{})

	_, err := handler.Handle(context.Background(), validCreateCommand())

	require.ErrorIs(t, err, logic.ErrEmailAlreadyExists)
}
