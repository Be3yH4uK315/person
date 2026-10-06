//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic"
	"person/internal/postgres"
)

func openIntegrationRepository(t *testing.T) (*dbr.Connection, *PersonRepository, *dbr.Session) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	connection, err := dbr.Open("pgx", dsn, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, connection.PingContext(context.Background()))
	return connection, NewPersonRepository(), connection.NewSession(nil)
}

func runInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return postgres.NewUnitOfWork().Do(ctx, fn)
}

func newIntegrationPerson(t *testing.T, email string) *app.Person {
	t.Helper()
	person, err := app.NewPerson(app.PersonDetails{
		Email: email, Phone: "+79990000000", FirstName: "Integration", LastName: "Test",
	})
	require.NoError(t, err)
	return person
}

func TestPersonRepository_OptimisticLocking(t *testing.T) {
	_, repo, session := openIntegrationRepository(t)
	person := newIntegrationPerson(t, "integration-locking@example.com")
	ctx := postgres.WithSession(context.Background(), session)

	defer func() { _, _ = session.DeleteFrom("persons").Where("id = ?", person.ID()).ExecContext(ctx) }()
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, person)
	}))

	loaded, err := repo.Get(ctx, person.ID())
	require.NoError(t, err)
	require.Equal(t, int64(1), loaded.Version())
	require.NoError(t, loaded.Replace(app.PersonDetails{
		Email: "integration-updated@example.com", Phone: "+79991111111", FirstName: "Updated", LastName: "Test",
	}))
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Update(tx, loaded, loaded.Version())
	}))
	require.ErrorIs(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Update(tx, loaded, loaded.Version())
	}), logic.ErrConcurrencyConflict)
}

func TestPersonRepository_DeleteRejectsStaleVersion(t *testing.T) {
	_, repo, session := openIntegrationRepository(t)
	person := newIntegrationPerson(t, "integration-delete@example.com")
	ctx := postgres.WithSession(context.Background(), session)

	defer func() { _, _ = session.DeleteFrom("persons").Where("id = ?", person.ID()).ExecContext(ctx) }()
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, person)
	}))
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Delete(tx, person.ID(), 1)
	}))
	// Recreate the aggregate for the stale-version guard.
	person = newIntegrationPerson(t, "integration-delete-stale@example.com")
	defer func() { _, _ = session.DeleteFrom("persons").Where("id = ?", person.ID()).ExecContext(ctx) }()
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, person)
	}))
	require.NoError(t, person.Replace(app.PersonDetails{
		Email: "integration-delete-stale-updated@example.com", Phone: "+79991111111", FirstName: "Updated", LastName: "Test",
	}))
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Update(tx, person, person.Version())
	}))
	require.ErrorIs(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Delete(tx, person.ID(), 1)
	}), logic.ErrConcurrencyConflict)
}

func TestPersonRepository_RejectsDuplicateEmail(t *testing.T) {
	_, repo, session := openIntegrationRepository(t)
	first := newIntegrationPerson(t, "integration-unique@example.com")
	second := newIntegrationPerson(t, " INTEGRATION-UNIQUE@example.com ")
	ctx := postgres.WithSession(context.Background(), session)

	defer func() {
		_, _ = session.DeleteFrom("persons").Where("id IN (?, ?)", first.ID(), second.ID()).ExecContext(ctx)
	}()
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, first)
	}))
	require.ErrorIs(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, second)
	}), logic.ErrEmailAlreadyExists)
}

func TestPersonRepository_ConcurrentUpdate_AllowsSingleWinner(t *testing.T) {
	connection, repo, session := openIntegrationRepository(t)
	person := newIntegrationPerson(t, "integration-concurrent@example.com")
	ctx := postgres.WithSession(context.Background(), session)

	defer func() { _, _ = session.DeleteFrom("persons").Where("id = ?", person.ID()).ExecContext(ctx) }()
	require.NoError(t, runInTransaction(ctx, func(tx context.Context) error {
		return repo.Create(tx, person)
	}))

	first, err := repo.Get(ctx, person.ID())
	require.NoError(t, err)
	second, err := repo.Get(ctx, person.ID())
	require.NoError(t, err)
	require.Equal(t, first.Version(), second.Version())

	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		local := connection.NewSession(nil)
		localCtx := postgres.WithSession(context.Background(), local)
		if err := first.Replace(app.PersonDetails{Email: "winner-a@example.com", Phone: "+79991111111", FirstName: "A", LastName: "Test"}); err != nil {
			results <- err
			return
		}
		results <- runInTransaction(localCtx, func(tx context.Context) error {
			return repo.Update(tx, first, first.Version())
		})
	}()
	go func() {
		defer wg.Done()
		local := connection.NewSession(nil)
		localCtx := postgres.WithSession(context.Background(), local)
		if err := second.Replace(app.PersonDetails{Email: "winner-b@example.com", Phone: "+79992222222", FirstName: "B", LastName: "Test"}); err != nil {
			results <- err
			return
		}
		results <- runInTransaction(localCtx, func(tx context.Context) error {
			return repo.Update(tx, second, second.Version())
		})
	}()
	wg.Wait()
	close(results)

	var success, conflicts int
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, logic.ErrConcurrencyConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent update error: %v", err)
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflicts)
}

func TestPersonRepository_DeleteMissingPersonReturnsNotFound(t *testing.T) {
	_, repo, session := openIntegrationRepository(t)
	ctx := postgres.WithSession(context.Background(), session)

	err := runInTransaction(ctx, func(tx context.Context) error {
		return repo.Delete(tx, uuid.New(), 1)
	})

	require.ErrorIs(t, err, logic.ErrPersonNotFound)
}

func TestPersonRepository_DeleteRemovesLegacyRowThatFailsFieldRules(t *testing.T) {
	_, repo, session := openIntegrationRepository(t)
	ctx := postgres.WithSession(context.Background(), session)
	id := uuid.New()
	defer func() { _, _ = session.DeleteFrom("persons").Where("id = ?", id).ExecContext(ctx) }()

	_, err := session.InsertInto("persons").
		Columns("id", "email", "phone", "first_name", "last_name", "created_at", "updated_at", "version").
		Values(id, "legacy-"+id.String(), "+79990000000", "Legacy", "Row", time.Now().UTC(), time.Now().UTC(), int64(1)).
		ExecContext(ctx)
	require.NoError(t, err)

	err = runInTransaction(ctx, func(tx context.Context) error {
		return repo.Delete(tx, id, 1)
	})

	require.NoError(t, err)
}

func TestPersonRepository_ConcurrentCreateEnforcesEmailUniqueness(t *testing.T) {
	connection, repo, _ := openIntegrationRepository(t)
	email := "integration-concurrent-unique-" + uuid.NewString() + "@example.com"
	first := newIntegrationPerson(t, email)
	second := newIntegrationPerson(t, " "+email+" ")
	t.Cleanup(func() {
		_, _ = connection.NewSession(nil).DeleteFrom("persons").Where("lower(btrim(email)) = ?", email).ExecContext(context.Background())
	})
	start := make(chan struct{})
	results := make(chan error, 2)

	for _, person := range []*app.Person{first, second} {
		person := person
		go func() {
			<-start
			ctx := postgres.WithSession(context.Background(), connection.NewSession(nil))
			results <- runInTransaction(ctx, func(tx context.Context) error { return repo.Create(tx, person) })
		}()
	}
	close(start)
	firstErr, secondErr := <-results, <-results

	successes := 0
	duplicates := 0
	for _, err := range []error{firstErr, secondErr} {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, logic.ErrEmailAlreadyExists):
			duplicates++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, duplicates)
}
