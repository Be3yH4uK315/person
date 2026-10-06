package commands

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/logic"
)

func TestDeletePersonHandler_PassesIDAndVersionToRepository(t *testing.T) {
	id := uuid.New()
	repository := &repositoryMock{}
	handler := NewDeletePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), DeletePersonCommand{ID: id, Version: 4})

	require.NoError(t, err)
	require.True(t, repository.deleted)
	require.Equal(t, id, repository.deleteID)
	require.EqualValues(t, 4, repository.deleteVersion)
}

func TestDeletePersonHandler_DoesNotLoadAggregate(t *testing.T) {
	repository := &repositoryMock{}
	handler := NewDeletePersonHandler(repository, uowMock{})

	err := handler.Handle(context.Background(), DeletePersonCommand{ID: uuid.New(), Version: 1})

	require.NoError(t, err)
	require.False(t, repository.getCalled)
}

func TestDeletePersonHandler_PropagatesConflictAndNotFound(t *testing.T) {
	for _, wantErr := range []error{logic.ErrConcurrencyConflict, logic.ErrPersonNotFound} {
		var rolledBack bool
		handler := NewDeletePersonHandler(&repositoryMock{deleteErr: wantErr}, uowMock{rolledBack: &rolledBack})

		err := handler.Handle(context.Background(), DeletePersonCommand{ID: uuid.New(), Version: 1})

		require.ErrorIs(t, err, wantErr)
		require.True(t, rolledBack)
	}
}

func TestDeletePersonHandler_RejectsInvalidCommand(t *testing.T) {
	handler := NewDeletePersonHandler(&repositoryMock{}, uowMock{})

	require.Error(t, handler.Handle(context.Background(), DeletePersonCommand{}))
	require.Error(t, handler.Handle(context.Background(), DeletePersonCommand{ID: uuid.New()}))
}
