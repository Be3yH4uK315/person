package commands

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic"
)

func storedPerson(t *testing.T, version int64) *app.Person {
	t.Helper()
	person, err := app.ReconstructPerson(app.PersonSnapshot{
		ID: uuid.New(), Email: "a@example.com", Phone: "+79991234567", FirstName: "A", LastName: "B",
		Version: version,
	})
	require.NoError(t, err)
	return person
}

func TestUpdatePersonHandler_UpdatesMatchingVersion(t *testing.T) {
	person := storedPerson(t, 3)
	repository := &repositoryMock{person: person}
	handler := NewUpdatePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{
		ID: person.ID(), Version: 3, Email: "b@example.com", Phone: "+79997654321", FirstName: "C", LastName: "D",
	})

	require.NoError(t, err)
	require.True(t, repository.updated)
	require.EqualValues(t, 3, repository.updateVersion)
	require.Equal(t, "b@example.com", person.Email())
}

func TestUpdatePersonHandler_RejectsStaleVersionBeforeMutation(t *testing.T) {
	person := storedPerson(t, 2)
	repository := &repositoryMock{person: person}
	handler := NewUpdatePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{
		ID: person.ID(), Version: 1, Email: "b@example.com", Phone: "+79991234568", FirstName: "C", LastName: "D",
	})

	require.ErrorIs(t, err, logic.ErrConcurrencyConflict)
	require.False(t, repository.updated)
	require.Equal(t, "a@example.com", person.Email())
}

func TestUpdatePersonHandler_RejectsNilPersonWithoutError(t *testing.T) {
	repository := &repositoryMock{}
	handler := NewUpdatePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{
		ID: uuid.New(), Version: 1, Email: "b@example.com", Phone: "+79991234568", FirstName: "C", LastName: "D",
	})

	require.ErrorContains(t, err, "repository returned nil person without error")
	require.True(t, repository.getCalled)
	require.False(t, repository.updated)
}

func TestUpdatePersonHandler_ReturnsNotFound(t *testing.T) {
	handler := NewUpdatePersonHandler(&repositoryMock{getErr: logic.ErrPersonNotFound}, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{
		ID: uuid.New(), Version: 1, Email: "b@example.com", Phone: "+79991234568", FirstName: "C", LastName: "D",
	})

	require.ErrorIs(t, err, logic.ErrPersonNotFound)
}

func TestUpdatePersonHandler_RejectsInvalidDataWithoutUpdating(t *testing.T) {
	person := storedPerson(t, 1)
	repository := &repositoryMock{person: person}
	handler := NewUpdatePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{ID: person.ID(), Version: 1, Email: "bad"})

	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.False(t, repository.updated)
}

func TestUpdatePersonHandler_CorrectsLegacyRow(t *testing.T) {
	legacy, err := app.ReconstructPerson(app.PersonSnapshot{
		ID: uuid.New(), Email: "legacy", Phone: "12+34", FirstName: "A", LastName: "B",
		Version: 1,
	})
	require.NoError(t, err)
	repository := &repositoryMock{person: legacy}
	handler := NewUpdatePersonHandler(repository, uowMock{})

	err = handler.Handle(context.Background(), UpdatePersonCommand{
		ID: legacy.ID(), Version: 1, Email: "fixed@example.com", Phone: "+79991234567", FirstName: "A", LastName: "B",
	})

	require.NoError(t, err)
	require.True(t, repository.updated)
}

func TestUpdatePersonHandler_RejectsInvalidCommand(t *testing.T) {
	handler := NewUpdatePersonHandler(&repositoryMock{}, uowMock{})

	err := handler.Handle(context.Background(), UpdatePersonCommand{})

	require.Error(t, err)
}
