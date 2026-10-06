package readers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gocraft/dbr/v2"
	"github.com/google/uuid"

	"person/internal/logic"
	"person/internal/logic/ports"
	"person/internal/postgres"
	"person/internal/postgres/model"
)

type PersonReader struct{}

func NewPersonReader() *PersonReader { return &PersonReader{} }

func (r *PersonReader) Get(ctx context.Context, id uuid.UUID) (ports.PersonView, error) {
	runner, err := postgres.Runner(ctx)
	if err != nil {
		return ports.PersonView{}, err
	}
	var m model.Person
	err = runner.Select(model.PersonColumns).From("persons").Where("id = ?", id).LoadOneContext(ctx, &m)
	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return ports.PersonView{}, logic.ErrPersonNotFound
		}
		return ports.PersonView{}, fmt.Errorf("get person view: %w", err)
	}
	return toView(m), nil
}

func (r *PersonReader) List(ctx context.Context, query ports.ListPersonsQuery) (ports.PersonListView, error) {
	limit, ok := nonNegativeUint64(query.Limit)
	if !ok {
		return ports.PersonListView{}, fmt.Errorf("invalid pagination: limit must not be negative")
	}
	offset, ok := nonNegativeUint64(query.Offset)
	if !ok {
		return ports.PersonListView{}, fmt.Errorf("invalid pagination: offset must not be negative")
	}
	runner, err := postgres.Runner(ctx)
	if err != nil {
		return ports.PersonListView{}, err
	}

	search := query.Search
	where := ""
	args := []interface{}{}
	if search != "" {
		pattern := "%" + escapeLikePattern(search) + "%"
		where = "(first_name ILIKE ? ESCAPE '\\' OR last_name ILIKE ? ESCAPE '\\' OR email ILIKE ? ESCAPE '\\' OR phone ILIKE ? ESCAPE '\\')"
		args = []interface{}{pattern, pattern, pattern, pattern}
	}

	countStmt := runner.Select("COUNT(*) AS total").From("persons")
	if where != "" {
		countStmt = countStmt.Where(where, args...)
	}
	total, err := countStmt.ReturnInt64Context(ctx)
	if err != nil {
		return ports.PersonListView{}, fmt.Errorf("list persons count: %w", err)
	}

	selectStmt := runner.Select(model.PersonColumns).From("persons").OrderBy("created_at DESC, id ASC").Limit(limit).Offset(offset)
	if where != "" {
		selectStmt = selectStmt.Where(where, args...)
	}
	var rows []model.Person
	if _, err := selectStmt.LoadContext(ctx, &rows); err != nil {
		return ports.PersonListView{}, fmt.Errorf("list persons: %w", err)
	}
	items := make([]ports.PersonView, 0, len(rows))
	for _, row := range rows {
		items = append(items, toView(row))
	}
	return ports.PersonListView{Items: items, Total: int(total)}, nil
}

func nonNegativeUint64(value int) (uint64, bool) {
	if value < 0 {
		return 0, false
	}
	return uint64(value), true
}

func escapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(value, "_", "\\_")
}

func toView(m model.Person) ports.PersonView {
	return ports.PersonView{
		ID:        m.ID,
		Email:     m.Email,
		Phone:     m.Phone,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		CreatedAt: m.CreatedAt.UTC(),
		UpdatedAt: m.UpdatedAt.UTC(),
		Version:   m.Version,
	}
}
