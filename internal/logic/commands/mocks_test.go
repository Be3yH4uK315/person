package commands

import (
	"context"

	"github.com/google/uuid"

	"person/internal/app"
)

type repositoryMock struct {
	person *app.Person

	created       bool
	createdPerson *app.Person
	updated       bool
	updatePerson  *app.Person
	updateVersion int64
	deleted       bool
	deleteID      uuid.UUID
	deleteVersion int64

	getCalled bool
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *repositoryMock) Get(context.Context, uuid.UUID) (*app.Person, error) {
	m.getCalled = true
	return m.person, m.getErr
}

func (m *repositoryMock) Create(_ context.Context, person *app.Person) error {
	m.created = true
	m.createdPerson = person
	return m.createErr
}

func (m *repositoryMock) Update(_ context.Context, person *app.Person, version int64) error {
	m.updated = true
	m.updatePerson = person
	m.updateVersion = version
	return m.updateErr
}

func (m *repositoryMock) Delete(_ context.Context, id uuid.UUID, version int64) error {
	m.deleted = true
	m.deleteID = id
	m.deleteVersion = version
	return m.deleteErr
}

type uowMock struct {
	committed  *bool
	rolledBack *bool
}

func (u uowMock) Do(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	if err == nil && u.committed != nil {
		*u.committed = true
	}
	if err != nil && u.rolledBack != nil {
		*u.rolledBack = true
	}
	return err
}
