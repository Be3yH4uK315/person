package queries

import (
	"context"
	"strings"

	"person/internal/logic"
	"person/internal/logic/ports"
)

type ListPersonsQueryHandler struct {
	reader ports.PersonReader
}

func NewListPersonsQueryHandler(reader ports.PersonReader) *ListPersonsQueryHandler {
	return &ListPersonsQueryHandler{reader: reader}
}

func (h *ListPersonsQueryHandler) Handle(ctx context.Context, query ports.ListPersonsQuery) (ports.PersonListView, error) {
	query.Search = strings.TrimSpace(query.Search)
	if err := logic.ValidateListQuery(query); err != nil {
		return ports.PersonListView{}, err
	}
	return h.reader.List(ctx, query)
}
