package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"person/internal/app"
	"person/internal/logic"
	"person/internal/postgres"
	"person/internal/postgres/mapper"
	"person/internal/postgres/model"
)

const emailUniqueConstraint = "uq_persons_email_normalized"

type PersonRepository struct{}

func NewPersonRepository() *PersonRepository { return &PersonRepository{} }

func (r *PersonRepository) Get(ctx context.Context, id uuid.UUID) (*app.Person, error) {
	runner, err := postgres.Runner(ctx)
	if err != nil {
		return nil, err
	}
	var m model.Person
	err = runner.Select(model.PersonColumns).From("persons").Where("id = ?", id).LoadOneContext(ctx, &m)
	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return nil, logic.ErrPersonNotFound
		}
		return nil, fmt.Errorf("get person: %w", err)
	}
	return mapper.ToDomain(m)
}

func (r *PersonRepository) Create(ctx context.Context, person *app.Person) error {
	if person == nil {
		return fmt.Errorf("create person: person is nil")
	}
	runner, err := postgres.TransactionRunner(ctx)
	if err != nil {
		return err
	}
	m := mapper.ToDB(person)
	now := time.Now().UTC()
	_, err = runner.InsertInto("persons").
		Columns("id", "email", "phone", "first_name", "last_name", "created_at", "updated_at", "version").
		Values(m.ID, m.Email, m.Phone, m.FirstName, m.LastName, now, now, m.Version).
		ExecContext(ctx)
	if err != nil {
		return classifyWriteError("create person", err)
	}
	return nil
}

func (r *PersonRepository) Update(ctx context.Context, person *app.Person, expectedVersion int64) error {
	if person == nil {
		return fmt.Errorf("update person: person is nil")
	}
	if expectedVersion == math.MaxInt64 {
		return fmt.Errorf("update person: version overflow")
	}
	runner, err := postgres.TransactionRunner(ctx)
	if err != nil {
		return err
	}
	m := mapper.ToDB(person)
	now := time.Now().UTC()
	result, err := runner.Update("persons").SetMap(map[string]interface{}{
		"email": m.Email, "phone": m.Phone, "first_name": m.FirstName, "last_name": m.LastName,
		"updated_at": now, "version": expectedVersion + 1,
	}).Where("id = ? AND version = ?", m.ID, expectedVersion).ExecContext(ctx)
	if err != nil {
		return classifyWriteError("update person", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update person rows affected: %w", err)
	}
	if affected != 1 {
		return logic.ErrConcurrencyConflict
	}
	return nil
}

func (r *PersonRepository) Delete(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	runner, err := postgres.TransactionRunner(ctx)
	if err != nil {
		return err
	}
	result, err := runner.DeleteFrom("persons").Where("id = ? AND version = ?", id, expectedVersion).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete person: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete person rows affected: %w", err)
	}
	if affected != 1 {
		return classifyDeleteMiss(ctx, runner, id)
	}
	return nil
}

func classifyDeleteMiss(ctx context.Context, runner dbr.SessionRunner, id uuid.UUID) error {
	_, err := runner.Select("version").From("persons").Where("id = ?", id).ReturnInt64Context(ctx)
	if errors.Is(err, dbr.ErrNotFound) {
		return logic.ErrPersonNotFound
	}
	if err != nil {
		return fmt.Errorf("check person existence: %w", err)
	}
	return logic.ErrConcurrencyConflict
}

func classifyWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == emailUniqueConstraint {
		return logic.ErrEmailAlreadyExists
	}
	return fmt.Errorf("%s: %w", operation, err)
}
