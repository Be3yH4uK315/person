//go:build integration

package readers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic/ports"
	"person/internal/postgres"
)

func TestPersonReader_ListSearch(t *testing.T) {
	dsn := testingDatabaseURL(t)
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	connection, err := dbr.Open("pgx", dsn, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	session := connection.NewSession(nil)
	ctx := postgres.WithSession(context.Background(), session)
	require.NoError(t, connection.PingContext(ctx))

	person, err := app.NewPerson(app.PersonDetails{
		Email: "integration-search@example.com", Phone: "79991234567", FirstName: "Searchable", LastName: "Reader",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = session.DeleteFrom("persons").Where("id = ?", person.ID()).ExecContext(ctx) })

	_, err = session.InsertInto("persons").Columns("id", "email", "phone", "first_name", "last_name", "created_at", "updated_at", "version").Values(
		person.ID(), person.Email(), person.Phone(), person.FirstName(), person.LastName(), time.Now().UTC(), time.Now().UTC(), int64(1),
	).ExecContext(ctx)
	require.NoError(t, err)

	result, err := NewPersonReader().List(ctx, ports.ListPersonsQuery{Limit: 20, Search: "Searchable"})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, person.ID(), result.Items[0].ID)
}

func testingDatabaseURL(t *testing.T) string {
	t.Helper()
	return os.Getenv("TEST_DATABASE_URL")
}

func TestPersonReader_SearchTreatsLikeWildcardsAsLiterals(t *testing.T) {
	dsn := testingDatabaseURL(t)
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	connection, err := dbr.Open("pgx", dsn, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	session := connection.NewSession(nil)
	ctx := postgres.WithSession(context.Background(), session)
	require.NoError(t, connection.PingContext(ctx))

	tag := uuid.NewString()[:8]
	percent, percentDecoy := "Pct%"+tag, "PctX"+tag
	underscore, underscoreDecoy := "Und_"+tag, "UndX"+tag
	backslash, backslashDecoy := `Back\`+tag, "BackX"+tag

	names := []string{percent, percentDecoy, underscore, underscoreDecoy, backslash, backslashDecoy}
	for index, name := range names {
		id := uuid.New()
		t.Cleanup(func() { _, _ = session.DeleteFrom("persons").Where("id = ?", id).ExecContext(ctx) })
		_, err := session.InsertInto("persons").
			Columns("id", "email", "phone", "first_name", "last_name", "created_at", "updated_at", "version").
			Values(id, fmt.Sprintf("wild-%s-%d@example.com", tag, index), "+79990000000", name, "Wild", time.Now().UTC(), time.Now().UTC(), int64(1)).
			ExecContext(ctx)
		require.NoError(t, err)
	}

	for _, term := range []string{percent, underscore, backslash} {
		result, err := NewPersonReader().List(ctx, ports.ListPersonsQuery{Limit: 20, Search: term})
		require.NoError(t, err, term)
		require.Equal(t, 1, result.Total, term)
		require.Len(t, result.Items, 1, term)
		require.Equal(t, term, result.Items[0].FirstName)
	}
}
