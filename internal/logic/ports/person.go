package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"person/internal/app"
)

type PersonRepository interface {
	Get(context.Context, uuid.UUID) (*app.Person, error)
	Create(context.Context, *app.Person) error
	Update(context.Context, *app.Person, int64) error
	Delete(context.Context, uuid.UUID, int64) error
}

type PersonReader interface {
	Get(context.Context, uuid.UUID) (PersonView, error)
	List(context.Context, ListPersonsQuery) (PersonListView, error)
}

type UnitOfWork interface {
	Do(context.Context, func(context.Context) error) error
}

type ListPersonsQuery struct {
	Limit  int
	Offset int
	Search string
}

type PersonView struct {
	ID        uuid.UUID
	Email     string
	Phone     string
	FirstName string
	LastName  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int64
}

type PersonListView struct {
	Items []PersonView
	Total int
}
