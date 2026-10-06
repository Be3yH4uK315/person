package queries

import (
	"context"

	"github.com/google/uuid"

	"person/internal/logic"
	"person/internal/logic/ports"
)

type GetPersonQuery struct {
	ID uuid.UUID
}

type GetPersonQueryHandler struct {
	reader ports.PersonReader
}

func NewGetPersonQueryHandler(reader ports.PersonReader) *GetPersonQueryHandler {
	return &GetPersonQueryHandler{reader: reader}
}

func (h *GetPersonQueryHandler) Handle(ctx context.Context, query GetPersonQuery) (ports.PersonView, error) {
	if err := logic.ValidateID(query.ID); err != nil {
		return ports.PersonView{}, err
	}
	return h.reader.Get(ctx, query.ID)
}
