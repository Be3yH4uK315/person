package commands

import (
	"context"

	"github.com/google/uuid"

	"person/internal/logic"
	"person/internal/logic/ports"
)

type DeletePersonCommand struct {
	ID      uuid.UUID
	Version int64
}

type DeletePersonHandler struct {
	repository ports.PersonRepository
	uow        ports.UnitOfWork
}

func NewDeletePersonHandler(repository ports.PersonRepository, uow ports.UnitOfWork) *DeletePersonHandler {
	return &DeletePersonHandler{repository: repository, uow: uow}
}

func (h *DeletePersonHandler) Handle(ctx context.Context, command DeletePersonCommand) error {
	if err := logic.ValidateMutation(command.ID, command.Version); err != nil {
		return err
	}
	return h.uow.Do(ctx, func(tx context.Context) error {
		return h.repository.Delete(tx, command.ID, command.Version)
	})
}
