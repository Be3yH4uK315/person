package app

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewPerson_InitializesDomainManagedState(t *testing.T) {
	person, err := NewPerson(PersonDetails{
		Email: "a@example.com", Phone: "79991234567", FirstName: "A", LastName: "B",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, person.ID())
	require.EqualValues(t, 1, person.Version())
}

func TestPerson_ReplaceUpdatesFieldsWithoutChangingVersion(t *testing.T) {
	person, err := NewPerson(PersonDetails{
		Email:     "a@example.com",
		Phone:     "79991234567",
		FirstName: "A",
		LastName:  "B",
	})
	require.NoError(t, err)
	err = person.Replace(PersonDetails{
		Email:     "b@example.com",
		Phone:     "79991234568",
		FirstName: "C",
		LastName:  "D",
	})
	require.NoError(t, err)
	require.Equal(t, "b@example.com", person.Email())
	require.EqualValues(t, 1, person.Version())
}

func TestReconstructPerson_RestoresVersion(t *testing.T) {
	person, err := ReconstructPerson(PersonSnapshot{
		ID:        uuid.New(),
		Email:     "a@example.com",
		Phone:     "79991234567",
		FirstName: "A",
		LastName:  "B",
		Version:   7,
	})
	require.NoError(t, err)
	require.EqualValues(t, 7, person.Version())
}

func TestNewPerson_NormalizesEmail(t *testing.T) {
	person, err := NewPerson(PersonDetails{
		Email: "  JOHN@Example.COM  ", Phone: "+79990000000", FirstName: "John", LastName: "Doe",
	})
	require.NoError(t, err)
	require.Equal(t, "john@example.com", person.Email())
}

func TestNewPerson_RejectsInvalidPhone(t *testing.T) {
	_, err := NewPerson(PersonDetails{
		Email: "john@example.com", Phone: "abc", FirstName: "John", LastName: "Doe",
	})
	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestNewPerson_RejectsTooLongName(t *testing.T) {
	_, err := NewPerson(PersonDetails{
		Email:     "john@example.com",
		Phone:     "79991234567",
		FirstName: strings.Repeat("a", 101),
		LastName:  "Doe",
	})
	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestNewPerson_ReportsAllValidationErrors(t *testing.T) {
	_, err := NewPerson(PersonDetails{})

	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, map[string]string{
		"email":     "is required",
		"phone":     "is required",
		"firstName": "is required",
		"lastName":  "is required",
	}, validation.Fields)
}

func TestPerson_ValidationDetailsStayInDomain(t *testing.T) {
	_, err := NewPerson(PersonDetails{
		Email: "not-an-email", Phone: "abc", FirstName: strings.Repeat("a", 101), LastName: "Doe",
	})

	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "must be a valid email", validation.Fields["email"])
	require.Equal(t, "must match +?[0-9]{7,15}", validation.Fields["phone"])
	require.Equal(t, "must not exceed 100 characters", validation.Fields["firstName"])
}

func TestPerson_ReplaceValidatesBeforeMutation(t *testing.T) {
	person, err := NewPerson(PersonDetails{
		Email: "john@example.com", Phone: "79991234567", FirstName: "John", LastName: "Doe",
	})
	require.NoError(t, err)

	beforeEmail := person.Email()
	beforePhone := person.Phone()
	beforeFirstName := person.FirstName()
	beforeLastName := person.LastName()
	err = person.Replace(PersonDetails{
		Email: "not-an-email", Phone: "abc", FirstName: "", LastName: "Doe",
	})

	require.Error(t, err)
	require.Equal(t, beforeEmail, person.Email())
	require.Equal(t, beforePhone, person.Phone())
	require.Equal(t, beforeFirstName, person.FirstName())
	require.Equal(t, beforeLastName, person.LastName())
}

func TestNewPerson_RejectsMalformedPhone(t *testing.T) {
	for _, phone := range []string{"+++1", "12+34", "123456", "1234567890123456"} {
		_, err := NewPerson(PersonDetails{
			Email: "john@example.com", Phone: phone, FirstName: "John", LastName: "Doe",
		})
		require.Error(t, err)
		var validation *ValidationError
		require.ErrorAs(t, err, &validation)
		require.Equal(t, "must match +?[0-9]{7,15}", validation.Fields["phone"])
	}
}

func TestNewPerson_RejectsControlCharactersInNames(t *testing.T) {
	_, err := NewPerson(PersonDetails{
		Email: "john@example.com", Phone: "79991234567", FirstName: "John\x00Doe", LastName: "Doe",
	})

	require.Error(t, err)
	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "must not contain control characters", validation.Fields["firstName"])
}

func TestReconstructPerson_AllowsRepairingLegacyFieldValues(t *testing.T) {
	person, err := ReconstructPerson(PersonSnapshot{
		ID: uuid.New(), Email: "not-an-email", Phone: "12+34", FirstName: "John", LastName: "Doe",
		Version: 1,
	})

	require.NoError(t, err)
	err = person.Replace(PersonDetails{
		Email: "fixed@example.com", Phone: "+79991234567", FirstName: "John", LastName: "Doe",
	})
	require.NoError(t, err)
	require.Equal(t, "fixed@example.com", person.Email())
}

func TestReconstructPerson_RejectsBrokenStructuralState(t *testing.T) {
	valid := PersonSnapshot{
		ID:        uuid.New(),
		Email:     "john@example.com",
		Phone:     "+79991234567",
		FirstName: "John",
		LastName:  "Doe",
		Version:   1,
	}
	for name, mutate := range map[string]func(*PersonSnapshot){
		"nil id":       func(s *PersonSnapshot) { s.ID = uuid.Nil },
		"zero version": func(s *PersonSnapshot) { s.Version = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := valid
			mutate(&snapshot)

			_, err := ReconstructPerson(snapshot)

			require.ErrorIs(t, err, ErrCorruptedPerson)
		})
	}
}

func TestPerson_ReplaceNilReceiverReturnsInvalidState(t *testing.T) {
	var person *Person
	err := person.Replace(PersonDetails{})
	require.ErrorIs(t, err, ErrInvalidPersonState)
}

func TestPersonOperationsNormalizeDetailsConsistently(t *testing.T) {
	person, err := NewPerson(PersonDetails{
		Email: "  JOHN@Example.COM  ", Phone: "  +79991234567  ",
		FirstName: "  John  ", LastName: "  Doe  ",
	})
	require.NoError(t, err)
	require.Equal(t, "john@example.com", person.Email())
	require.Equal(t, "+79991234567", person.Phone())
	require.Equal(t, "John", person.FirstName())
	require.Equal(t, "Doe", person.LastName())

	err = person.Replace(PersonDetails{
		Email: "  JANE@Example.COM  ", Phone: "  +79991234568  ",
		FirstName: "  Jane  ", LastName: "  Smith  ",
	})
	require.NoError(t, err)
	require.Equal(t, "jane@example.com", person.Email())
	require.Equal(t, "+79991234568", person.Phone())
	require.Equal(t, "Jane", person.FirstName())
	require.Equal(t, "Smith", person.LastName())
}
