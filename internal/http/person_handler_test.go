package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/http/request"
	"person/internal/logic/commands"
	"person/internal/logic/ports"
	"person/internal/logic/queries"
)

type HandlerDependencies struct {
	Create personCreator
	Update personUpdater
	Delete personDeleter
	Get    personGetter
	List   personLister
}

func mustNewHandler(t *testing.T, deps HandlerDependencies) *Handler {
	t.Helper()
	if deps.Create == nil {
		deps.Create = &fakeCreator{}
	}
	if deps.Update == nil {
		deps.Update = &fakeUpdater{}
	}
	if deps.Delete == nil {
		deps.Delete = &fakeDeleter{}
	}
	if deps.Get == nil {
		deps.Get = &fakeGetter{}
	}
	if deps.List == nil {
		deps.List = &fakeLister{}
	}
	return NewHandler(deps.Create, deps.Update, deps.Delete, deps.Get, deps.List)
}

type fakeUpdater struct {
	called  bool
	command commands.UpdatePersonCommand
	err     error
}

func (f *fakeUpdater) Handle(_ context.Context, command commands.UpdatePersonCommand) error {
	f.called = true
	f.command = command
	return f.err
}

type fakeDeleter struct {
	called  bool
	command commands.DeletePersonCommand
	err     error
}

func (f *fakeDeleter) Handle(_ context.Context, command commands.DeletePersonCommand) error {
	f.called = true
	f.command = command
	return f.err
}

func TestBindBody_RejectsUnknownFields(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost, "/person",
		strings.NewReader(`{"email":"john@example.com","unknown":"value"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	var body request.CreatePersonRequest
	err := bindBody(ctx, &body)

	require.Error(t, err)
	var validation *app.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "must contain valid JSON", validation.Fields["body"])
}

func TestBindBody_RejectsTrailingJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost, "/person",
		strings.NewReader(`{"email":"john@example.com"} {}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	var body request.CreatePersonRequest
	err := bindBody(ctx, &body)

	require.Error(t, err)
}

func TestBindBody_RequiresJSONContentType(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost, "/person",
		strings.NewReader(`{"email":"john@example.com"}`),
	)
	req.Header.Set(echo.HeaderContentType, "text/plain")
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	var body request.CreatePersonRequest
	err := bindBody(ctx, &body)

	require.Error(t, err)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnsupportedMediaType, httpErr.Code)
}

func TestHandlerUpdate_DelegatesVersionValidationToUseCase(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut, "/person/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"email":"john@example.com","phone":"79991234567","firstName":"John","lastName":"Doe","version":0}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}})
	validationErr := app.NewValidationError(map[string]string{"version": "must be positive"})
	fake := &fakeUpdater{err: validationErr}
	h := mustNewHandler(t, HandlerDependencies{Update: fake})

	err := h.Update(ctx)

	require.ErrorIs(t, err, validationErr)
	require.True(t, fake.called)
	require.Zero(t, fake.command.Version)
}

func TestHandlerUpdate_DelegatesMissingVersionToUseCase(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut, "/person/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"email":"john@example.com","phone":"79991234567","firstName":"John","lastName":"Doe"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}})
	validationErr := app.NewValidationError(map[string]string{"version": "must be positive"})
	fake := &fakeUpdater{err: validationErr}
	h := mustNewHandler(t, HandlerDependencies{Update: fake})

	err := h.Update(ctx)

	require.ErrorIs(t, err, validationErr)
	require.True(t, fake.called)
	require.Zero(t, fake.command.Version)
}

func TestHandlerUpdate_PassesExpectedVersion(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut, "/person/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"email":"john@example.com","phone":"79991234567","firstName":"John","lastName":"Doe","version":7}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}})
	fake := &fakeUpdater{}
	h := mustNewHandler(t, HandlerDependencies{Update: fake})

	err := h.Update(ctx)

	require.NoError(t, err)
	require.True(t, fake.called)
	require.Equal(t, int64(7), fake.command.Version)
	require.Equal(t, http.StatusOK, resp.Code)
	require.JSONEq(t, `{"id":"550e8400-e29b-41d4-a716-446655440000","uri":"/person/550e8400-e29b-41d4-a716-446655440000"}`, resp.Body.String())
}

func TestHandlerDelete_RejectsEmptyBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete, "/person/550e8400-e29b-41d4-a716-446655440000", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}})
	fake := &fakeDeleter{}
	h := mustNewHandler(t, HandlerDependencies{Delete: fake})

	err := h.Delete(ctx)

	require.Error(t, err)
	require.False(t, fake.called)
}

func TestHandlerDelete_PassesExpectedVersion(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete, "/person/550e8400-e29b-41d4-a716-446655440000", strings.NewReader(`{"version":9}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}})
	fake := &fakeDeleter{}
	h := mustNewHandler(t, HandlerDependencies{Delete: fake})

	err := h.Delete(ctx)

	require.NoError(t, err)
	require.True(t, fake.called)
	require.Equal(t, int64(9), fake.command.Version)
	require.JSONEq(t, `{"id":"550e8400-e29b-41d4-a716-446655440000","uri":"/person/550e8400-e29b-41d4-a716-446655440000"}`, resp.Body.String())
}

type fakeCreator struct {
	called  bool
	command commands.CreatePersonCommand
	id      uuid.UUID
	err     error
}

func (f *fakeCreator) Handle(_ context.Context, command commands.CreatePersonCommand) (uuid.UUID, error) {
	f.called = true
	f.command = command
	return f.id, f.err
}

type fakeGetter struct {
	query queries.GetPersonQuery
	view  ports.PersonView
	err   error
}

func (f *fakeGetter) Handle(_ context.Context, query queries.GetPersonQuery) (ports.PersonView, error) {
	f.query = query
	return f.view, f.err
}

type fakeLister struct {
	query  ports.ListPersonsQuery
	result ports.PersonListView
	err    error
}

func (f *fakeLister) Handle(_ context.Context, query ports.ListPersonsQuery) (ports.PersonListView, error) {
	f.query = query
	return f.result, f.err
}

func TestHandlerCreate_MapsRequestToCommand(t *testing.T) {
	id := uuid.New()
	fake := &fakeCreator{id: id}
	h := mustNewHandler(t, HandlerDependencies{Create: fake})
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost, "/person", strings.NewReader(`{"email":"john@example.com","phone":"+79991234567","firstName":"John","lastName":"Doe"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := h.Create(ctx)

	require.NoError(t, err)
	require.True(t, fake.called)
	require.Equal(t, "john@example.com", fake.command.Email)
	require.Equal(t, "+79991234567", fake.command.Phone)
	require.Equal(t, http.StatusCreated, resp.Code)
	require.JSONEq(t, `{"id":"`+id.String()+`","uri":"/person/`+id.String()+`"}`, resp.Body.String())
}

func TestHandlerGet_MapsViewToResponse(t *testing.T) {
	id := uuid.New()
	now := time.Date(2026, time.October, 8, 7, 0, 0, 123, time.UTC)
	fake := &fakeGetter{view: ports.PersonView{
		ID: id, Email: "john@example.com", Phone: "+79991234567", FirstName: "John", LastName: "Doe",
		CreatedAt: now, UpdatedAt: now, Version: 4,
	}}
	h := mustNewHandler(t, HandlerDependencies{Get: fake})
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet, "/person/"+id.String(), nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)
	ctx.SetPath("/person/:id")
	ctx.SetPathValues(echo.PathValues{{Name: "id", Value: id.String()}})

	err := h.Get(ctx)

	require.NoError(t, err)
	require.Equal(t, id, fake.query.ID)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), `"version":4`)
	require.Contains(t, resp.Body.String(), now.Format(time.RFC3339Nano))
}

func TestHandlerList_RejectsInvalidLimitFromAlternateUseCase(t *testing.T) {
	h := mustNewHandler(t, HandlerDependencies{List: &fakeLister{}})
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/person?limit=0", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := h.List(ctx)

	require.ErrorContains(t, err, "invalid limit")
}

func TestHandlerList_PassesPaginationAndSearch(t *testing.T) {
	fake := &fakeLister{result: ports.PersonListView{Total: 45}}
	h := mustNewHandler(t, HandlerDependencies{List: fake})
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet, "/person?limit=20&offset=20&search=john", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := h.List(ctx)

	require.NoError(t, err)
	require.Equal(t, ports.ListPersonsQuery{Limit: 20, Offset: 20, Search: "john"}, fake.query)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), `"page":2`)
	require.Contains(t, resp.Body.String(), `"pages":3`)
}
