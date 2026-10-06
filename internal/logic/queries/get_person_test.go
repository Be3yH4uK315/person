package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic/ports"
)

type getPersonReaderMock struct {
	id     uuid.UUID
	result ports.PersonView
	err    error
}

func (m *getPersonReaderMock) Get(_ context.Context, id uuid.UUID) (ports.PersonView, error) {
	m.id = id
	return m.result, m.err
}

func (m *getPersonReaderMock) List(context.Context, ports.ListPersonsQuery) (ports.PersonListView, error) {
	return ports.PersonListView{}, nil
}

func TestGetPersonQueryHandler_PassesIDToReader(t *testing.T) {
	id := uuid.New()
	reader := &getPersonReaderMock{result: ports.PersonView{ID: id}}
	handler := NewGetPersonQueryHandler(reader)

	result, err := handler.Handle(context.Background(), GetPersonQuery{ID: id})

	require.NoError(t, err)
	require.Equal(t, id, reader.id)
	require.Equal(t, id, result.ID)
}

func TestGetPersonQueryHandler_RejectsNilID(t *testing.T) {
	handler := NewGetPersonQueryHandler(&getPersonReaderMock{})

	_, err := handler.Handle(context.Background(), GetPersonQuery{})

	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "must be a valid UUID", validation.Fields["id"])
}

func TestGetPersonQueryHandler_PropagatesReaderError(t *testing.T) {
	wantErr := errors.New("reader failure")
	handler := NewGetPersonQueryHandler(&getPersonReaderMock{err: wantErr})

	_, err := handler.Handle(context.Background(), GetPersonQuery{ID: uuid.New()})

	require.ErrorIs(t, err, wantErr)
}
