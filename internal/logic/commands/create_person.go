package commands

import (
	"context"

	"github.com/google/uuid"

	"person/internal/app"
	"person/internal/logic/ports"
)

type CreatePersonCommand struct {
	Email     string
	Phone     string
	FirstName string
	LastName  string
}

type CreatePersonHandler struct {
	repository ports.PersonRepository
	uow        ports.UnitOfWork
}

func NewCreatePersonHandler(repository ports.PersonRepository, uow ports.UnitOfWork) *CreatePersonHandler {
	return &CreatePersonHandler{repository: repository, uow: uow}
}

func (h *CreatePersonHandler) Handle(ctx context.Context, command CreatePersonCommand) (uuid.UUID, error) {
	person, err := app.NewPerson(app.PersonDetails{
		Email: command.Email, Phone: command.Phone,
		FirstName: command.FirstName, LastName: command.LastName,
	})
	if err != nil {
		return uuid.Nil, err
	}

	err = h.uow.Do(ctx, func(tx context.Context) error {
		return h.repository.Create(tx, person)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return person.ID(), nil
}
