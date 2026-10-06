package mapper

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/postgres/model"
)

func TestToDB_MapsAggregateState(t *testing.T) {
	id := uuid.New()
	person, err := app.ReconstructPerson(app.PersonSnapshot{
		ID: id, Email: "a@example.com", Phone: "+79990000000", FirstName: "John", LastName: "Doe",
		Version: 7,
	})
	require.NoError(t, err)

	got := ToDB(person)

	require.Equal(t, model.Person{
		ID: id, Email: "a@example.com", Phone: "+79990000000", FirstName: "John", LastName: "Doe",
		Version: 7,
	}, got)
}

func TestToDomain_RestoresAggregateState(t *testing.T) {
	id := uuid.New()

	person, err := ToDomain(model.Person{
		ID: id, Email: "a@example.com", Phone: "+79990000000", FirstName: "John", LastName: "Doe",
		Version: 7,
	})

	require.NoError(t, err)
	require.Equal(t, id, person.ID())
	require.Equal(t, int64(7), person.Version())
}
