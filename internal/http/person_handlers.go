package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"person/internal/app"
	"person/internal/http/request"
	"person/internal/http/response"
	"person/internal/logic"
	"person/internal/logic/commands"
	"person/internal/logic/ports"
	"person/internal/logic/queries"
)

type personCreator interface {
	Handle(context.Context, commands.CreatePersonCommand) (uuid.UUID, error)
}

type personUpdater interface {
	Handle(context.Context, commands.UpdatePersonCommand) error
}

type personDeleter interface {
	Handle(context.Context, commands.DeletePersonCommand) error
}

type personGetter interface {
	Handle(context.Context, queries.GetPersonQuery) (ports.PersonView, error)
}

type personLister interface {
	Handle(context.Context, ports.ListPersonsQuery) (ports.PersonListView, error)
}

type Handler struct {
	create personCreator
	update personUpdater
	delete personDeleter
	get    personGetter
	list   personLister
}

func NewHandler(create personCreator, update personUpdater, del personDeleter, get personGetter, list personLister) *Handler {
	return &Handler{create: create, update: update, delete: del, get: get, list: list}
}

func (h *Handler) Create(c *echo.Context) error {
	var body request.CreatePersonRequest
	if err := bindBody(c, &body); err != nil {
		return err
	}
	id, err := h.create.Handle(c.Request().Context(), commands.CreatePersonCommand{
		Email: body.Email, Phone: body.Phone, FirstName: body.FirstName, LastName: body.LastName,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, response.MutationResponse{ID: id.String(), URI: personURI(id)})
}

func (h *Handler) Get(c *echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	view, err := h.get.Handle(c.Request().Context(), queries.GetPersonQuery{ID: id})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(view))
}

func (h *Handler) List(c *echo.Context) error {
	limit, err := parseOptionalIntQuery(c, "limit", logic.DefaultLimit)
	if err != nil {
		return err
	}
	offset, err := parseOptionalIntQuery(c, "offset", 0)
	if err != nil {
		return err
	}
	search := c.QueryParam("search")
	result, err := h.list.Handle(c.Request().Context(), ports.ListPersonsQuery{
		Limit: limit, Offset: offset, Search: search,
	})
	if err != nil {
		return err
	}
	// ListPersonsQueryHandler validates limit before returning successfully. Keep
	// this guard as a safety net for alternate implementations of personLister.
	if limit < 1 {
		return errors.New("list handler returned success for an invalid limit")
	}
	page := offset/limit + 1
	pages := 0
	if result.Total > 0 {
		pages = 1 + (result.Total-1)/limit
	}
	return c.JSON(http.StatusOK, response.PersonListResponse{
		Items: toResponses(result.Items),
		Pagination: response.Pagination{
			Limit: limit, Offset: offset, Total: result.Total,
			Page: page, Pages: pages,
		},
	})
}

func (h *Handler) Update(c *echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body request.UpdatePersonRequest
	if err := bindBody(c, &body); err != nil {
		return err
	}
	if err := h.update.Handle(c.Request().Context(), commands.UpdatePersonCommand{
		ID: id, Version: body.Version,
		Email: body.Email, Phone: body.Phone, FirstName: body.FirstName, LastName: body.LastName,
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response.MutationResponse{ID: id.String(), URI: personURI(id)})
}

func (h *Handler) Delete(c *echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body request.DeletePersonRequest
	if err := bindBody(c, &body); err != nil {
		return err
	}
	if err := h.delete.Handle(c.Request().Context(), commands.DeletePersonCommand{
		ID:      id,
		Version: body.Version,
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response.MutationResponse{ID: id.String(), URI: personURI(id)})
}

// parseOptionalIntQuery parses syntax only; range validation belongs to the
// list query use case.
func parseOptionalIntQuery(c *echo.Context, name string, fallback int) (int, error) {
	value := c.QueryParam(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, app.NewValidationError(map[string]string{name: "must be an integer"})
	}
	return parsed, nil
}

func personURI(id uuid.UUID) string { return "/person/" + id.String() }

func bindBody(c *echo.Context, target any) error {
	contentType := c.Request().Header.Get(echo.HeaderContentType)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, "content type must be application/json")
	}

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "request body too large")
		}
		return app.NewValidationError(map[string]string{"body": "must contain valid JSON"})
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return app.NewValidationError(map[string]string{"body": "must contain exactly one JSON object"})
	}
	return nil
}

func parseID(c *echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, app.NewValidationError(map[string]string{"id": "must be a valid UUID"})
	}
	return id, nil
}

func toResponse(v ports.PersonView) response.PersonResponse {
	return response.PersonResponse{
		ID:        v.ID.String(),
		Email:     v.Email,
		Phone:     v.Phone,
		FirstName: v.FirstName,
		LastName:  v.LastName,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Version:   v.Version,
	}
}
func toResponses(items []ports.PersonView) []response.PersonResponse {
	result := make([]response.PersonResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toResponse(item))
	}
	return result
}
