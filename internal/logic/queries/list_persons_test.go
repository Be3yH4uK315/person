package queries

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic/ports"
)

type personReaderMock struct {
	called    bool
	listQuery ports.ListPersonsQuery
	result    ports.PersonListView
}

func (m *personReaderMock) Get(context.Context, uuid.UUID) (ports.PersonView, error) {
	return ports.PersonView{}, nil
}

func (m *personReaderMock) List(_ context.Context, query ports.ListPersonsQuery) (ports.PersonListView, error) {
	m.called = true
	m.listQuery = query
	return m.result, nil
}

func TestListPersonsQueryHandler_PassesQueryToReader(t *testing.T) {
	reader := &personReaderMock{result: ports.PersonListView{Total: 3}}
	handler := NewListPersonsQueryHandler(reader)

	result, err := handler.Handle(context.Background(), ports.ListPersonsQuery{
		Limit:  20,
		Offset: 40,
		Search: "john",
	})

	require.NoError(t, err)
	require.Equal(t, ports.ListPersonsQuery{Limit: 20, Offset: 40, Search: "john"}, reader.listQuery)
	require.Equal(t, 3, result.Total)
}

func TestListPersonsQueryHandler_RejectsInvalidQueryBeforeReader(t *testing.T) {
	reader := &personReaderMock{}
	handler := NewListPersonsQueryHandler(reader)
	_, err := handler.Handle(context.Background(), ports.ListPersonsQuery{Limit: 0, Offset: -1, Search: strings.Repeat("x", 129)})
	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.False(t, reader.called)
}
