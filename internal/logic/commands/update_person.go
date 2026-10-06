package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"person/internal/app"
	"person/internal/logic"
	"person/internal/logic/ports"
)

type UpdatePersonCommand struct {
	ID        uuid.UUID
	Version   int64
	Email     string
	Phone     string
	FirstName string
	LastName  string
}

type UpdatePersonHandler struct {
	repository ports.PersonRepository
	uow        ports.UnitOfWork
}

func NewUpdatePersonHandler(repository ports.PersonRepository, uow ports.UnitOfWork) *UpdatePersonHandler {
	return &UpdatePersonHandler{repository: repository, uow: uow}
}

func (h *UpdatePersonHandler) Handle(ctx context.Context, command UpdatePersonCommand) error {
	if err := logic.ValidateMutation(command.ID, command.Version); err != nil {
		return err
	}

	return h.uow.Do(ctx, func(tx context.Context) error {
		person, err := h.repository.Get(tx, command.ID)
		if err != nil {
			return err
		}
		if person == nil {
			return fmt.Errorf("repository returned nil person without error")
		}
		if person.Version() != command.Version {
			return logic.ErrConcurrencyConflict
		}
		if err := person.Replace(app.PersonDetails{
			Email: command.Email, Phone: command.Phone,
			FirstName: command.FirstName, LastName: command.LastName,
		}); err != nil {
			return err
		}
		return h.repository.Update(tx, person, command.Version)
	})
}
