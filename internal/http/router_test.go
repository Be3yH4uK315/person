package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"person/internal/app"
	"person/internal/logic"
	"person/internal/logic/ports"
	"person/internal/logic/queries"
)

func validRouterOptions() RouterOptions {
	return RouterOptions{
		MaxBodyBytes:       1024,
		ContextTimeout:     time.Second,
		HealthCheckTimeout: time.Second,
		ContextEnricher:    func(ctx context.Context) context.Context { return ctx },
		ReadinessCheck:     func(context.Context) error { return nil },
	}
}

func TestNewRouter_RejectsInvalidOptions(t *testing.T) {
	handler := mustNewHandler(t, HandlerDependencies{})
	logger := logrus.New()

	options := validRouterOptions()
	options.ContextTimeout = 0
	_, err := NewRouter(handler, logger, options)

	require.ErrorContains(t, err, "context timeout must be positive")
}

func TestRouter_HealthEndpoints(t *testing.T) {
	handler := mustNewHandler(t, HandlerDependencies{})
	logger := logrus.New()
	router, err := NewRouter(handler, logger, validRouterOptions())
	require.NoError(t, err)

	live := httptest.NewRecorder()
	router.ServeHTTP(live, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/live", nil))
	require.Equal(t, http.StatusOK, live.Code)
	require.JSONEq(t, `{"status":"ok"}`, live.Body.String())

	ready := httptest.NewRecorder()
	router.ServeHTTP(ready, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil))
	require.Equal(t, http.StatusOK, ready.Code)
	require.JSONEq(t, `{"status":"ok"}`, ready.Body.String())
}

func TestRouter_ReadinessFailureReturnsJSON503(t *testing.T) {
	handler := mustNewHandler(t, HandlerDependencies{})
	logger := logrus.New()
	options := validRouterOptions()
	options.ReadinessCheck = func(context.Context) error { return errors.New("database unavailable") }
	router, err := NewRouter(handler, logger, options)
	require.NoError(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil))

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	require.JSONEq(t, `{"status":"unavailable"}`, resp.Body.String())
}

func TestNewRouter_RejectsMissingReadinessCheck(t *testing.T) {
	options := validRouterOptions()
	options.ReadinessCheck = nil

	_, err := NewRouter(mustNewHandler(t, HandlerDependencies{}), logrus.New(), options)

	require.ErrorContains(t, err, "readiness check is required")
}

func newTestRouter(t *testing.T, deps HandlerDependencies, mutate func(*RouterOptions)) *echo.Echo {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	options := validRouterOptions()
	if mutate != nil {
		mutate(&options)
	}
	router, err := NewRouter(mustNewHandler(t, deps), logger, options)
	require.NoError(t, err)
	return router
}

func serve(router http.Handler, method, target, contentType, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, target, reader)
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

type errorBodyForTest struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

type errorEnvelopeForTest struct {
	Error     errorBodyForTest `json:"error"`
	RequestID string           `json:"requestId"`
}

func decodeError(t *testing.T, resp *httptest.ResponseRecorder) errorEnvelopeForTest {
	t.Helper()
	var envelope errorEnvelopeForTest
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope), resp.Body.String())
	return envelope
}

const validPersonJSON = `{"email":"john@example.com","phone":"+79991234567","firstName":"John","lastName":"Doe"}`

func TestRouter_MapsErrorsToHTTPContract(t *testing.T) {
	id := uuid.New().String()
	validationErr := app.NewValidationError(map[string]string{"email": "is required"})
	oversizedBody := `{"email":"` + strings.Repeat("a", 2048) + `"}`

	tests := []struct {
		name        string
		deps        HandlerDependencies
		method      string
		target      string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
	}{
		{
			name: "get not found", deps: HandlerDependencies{Get: &fakeGetter{err: logic.ErrPersonNotFound}},
			method: http.MethodGet, target: "/person/" + id,
			wantStatus: http.StatusNotFound, wantCode: "PERSON_NOT_FOUND",
		},
		{
			name: "get invalid id", deps: HandlerDependencies{Get: &fakeGetter{}},
			method: http.MethodGet, target: "/person/not-a-uuid",
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR",
		},
		{
			name: "update version conflict", deps: HandlerDependencies{Update: &fakeUpdater{err: logic.ErrConcurrencyConflict}},
			method: http.MethodPut, target: "/person/" + id, contentType: echo.MIMEApplicationJSON,
			body:       `{"email":"john@example.com","phone":"+79991234567","firstName":"John","lastName":"Doe","version":1}`,
			wantStatus: http.StatusConflict, wantCode: "CONCURRENCY_CONFLICT",
		},
		{
			name: "create duplicate email", deps: HandlerDependencies{Create: &fakeCreator{err: logic.ErrEmailAlreadyExists}},
			method: http.MethodPost, target: "/person", contentType: echo.MIMEApplicationJSON, body: validPersonJSON,
			wantStatus: http.StatusConflict, wantCode: "PERSON_EMAIL_ALREADY_EXISTS",
		},
		{
			name: "create domain validation", deps: HandlerDependencies{Create: &fakeCreator{err: validationErr}},
			method: http.MethodPost, target: "/person", contentType: echo.MIMEApplicationJSON, body: validPersonJSON,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR",
		},
		{
			name: "delete not found", deps: HandlerDependencies{Delete: &fakeDeleter{err: logic.ErrPersonNotFound}},
			method: http.MethodDelete, target: "/person/" + id, contentType: echo.MIMEApplicationJSON, body: `{"version":1}`,
			wantStatus: http.StatusNotFound, wantCode: "PERSON_NOT_FOUND",
		},
		{
			name: "unsupported media type", deps: HandlerDependencies{Create: &fakeCreator{}},
			method: http.MethodPost, target: "/person", contentType: "text/plain", body: validPersonJSON,
			wantStatus: http.StatusUnsupportedMediaType, wantCode: "UNSUPPORTED_MEDIA_TYPE",
		},
		{
			name: "payload too large", deps: HandlerDependencies{Create: &fakeCreator{}},
			method: http.MethodPost, target: "/person", contentType: echo.MIMEApplicationJSON, body: oversizedBody,
			wantStatus: http.StatusRequestEntityTooLarge, wantCode: "PAYLOAD_TOO_LARGE",
		},
		{
			name: "unknown route", deps: HandlerDependencies{},
			method: http.MethodGet, target: "/nope",
			wantStatus: http.StatusNotFound, wantCode: "ROUTE_NOT_FOUND",
		},
		{
			name: "method not allowed", deps: HandlerDependencies{},
			method: http.MethodPatch, target: "/person",
			wantStatus: http.StatusMethodNotAllowed, wantCode: "METHOD_NOT_ALLOWED",
		},
		{
			name: "unexpected error is not leaked", deps: HandlerDependencies{Get: &fakeGetter{err: errors.New("pq: password authentication failed")}},
			method: http.MethodGet, target: "/person/" + id,
			wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, tt.deps, nil)

			resp := serve(router, tt.method, tt.target, tt.contentType, tt.body)

			require.Equal(t, tt.wantStatus, resp.Code, resp.Body.String())
			if tt.wantStatus == http.StatusMethodNotAllowed {
				require.Contains(t, resp.Header().Get(echo.HeaderAllow), http.MethodGet)
				require.Contains(t, resp.Header().Get(echo.HeaderAllow), http.MethodPost)
			}
			envelope := decodeError(t, resp)
			require.Equal(t, tt.wantCode, envelope.Error.Code)
			require.NotEmpty(t, envelope.RequestID)
			require.Equal(t, resp.Header().Get(echo.HeaderXRequestID), envelope.RequestID)
			require.NotContains(t, resp.Body.String(), "password")
		})
	}
}

func TestRouter_ValidationDetailsAreReturned(t *testing.T) {
	validationErr := app.NewValidationError(map[string]string{"email": "is required"})
	router := newTestRouter(t, HandlerDependencies{Create: &fakeCreator{err: validationErr}}, nil)

	resp := serve(router, http.MethodPost, "/person", echo.MIMEApplicationJSON, validPersonJSON)

	require.Equal(t, map[string]string{"email": "is required"}, decodeError(t, resp).Error.Details)
}

func TestRouter_CRUDHappyPath(t *testing.T) {
	id := uuid.New()
	now := time.Date(2026, time.October, 8, 7, 0, 0, 0, time.UTC)
	view := ports.PersonView{
		ID: id, Email: "john@example.com", Phone: "+79991234567", FirstName: "John", LastName: "Doe",
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	router := newTestRouter(t, HandlerDependencies{
		Create: &fakeCreator{id: id},
		Get:    &fakeGetter{view: view},
		List:   &fakeLister{result: ports.PersonListView{Items: []ports.PersonView{view}, Total: 1}},
		Update: &fakeUpdater{},
		Delete: &fakeDeleter{},
	}, nil)

	created := serve(router, http.MethodPost, "/person/", echo.MIMEApplicationJSON, validPersonJSON)
	require.Equal(t, http.StatusCreated, created.Code)
	require.JSONEq(t, `{"id":"`+id.String()+`","uri":"/person/`+id.String()+`"}`, created.Body.String())
	require.NotEmpty(t, created.Header().Get(echo.HeaderXRequestID))

	got := serve(router, http.MethodGet, "/person/"+id.String(), "", "")
	require.Equal(t, http.StatusOK, got.Code)
	require.Contains(t, got.Body.String(), `"firstName":"John"`)
	require.Equal(t, "no-store", got.Header().Get("Cache-Control"))

	list := serve(router, http.MethodGet, "/person?limit=10", "", "")
	require.Equal(t, http.StatusOK, list.Code)
	require.Contains(t, list.Body.String(), `"total":1`)

	updated := serve(router, http.MethodPut, "/person/"+id.String(), echo.MIMEApplicationJSON,
		`{"email":"john@example.com","phone":"+79991234567","firstName":"John","lastName":"Smith","version":1}`)
	require.Equal(t, http.StatusOK, updated.Code)
	require.JSONEq(t, `{"id":"`+id.String()+`","uri":"/person/`+id.String()+`"}`, updated.Body.String())

	deleted := serve(router, http.MethodDelete, "/person/"+id.String(), echo.MIMEApplicationJSON, `{"version":2}`)
	require.Equal(t, http.StatusOK, deleted.Code)
	require.JSONEq(t, `{"id":"`+id.String()+`","uri":"/person/`+id.String()+`"}`, deleted.Body.String())
}

func TestRouter_RejectsInvalidListParameters(t *testing.T) {
	router := newTestRouter(t, HandlerDependencies{
		List: queries.NewListPersonsQueryHandler(emptyPersonReader{}),
	}, nil)

	for _, target := range []string{"/person?limit=0", "/person?limit=101", "/person?limit=abc", "/person?offset=-1"} {
		resp := serve(router, http.MethodGet, target, "", "")

		require.Equal(t, http.StatusBadRequest, resp.Code, target)
		require.Equal(t, "VALIDATION_ERROR", decodeError(t, resp).Error.Code, target)
	}
}

type emptyPersonReader struct{}

func (emptyPersonReader) Get(context.Context, uuid.UUID) (ports.PersonView, error) {
	return ports.PersonView{}, nil
}

func (emptyPersonReader) List(context.Context, ports.ListPersonsQuery) (ports.PersonListView, error) {
	return ports.PersonListView{}, nil
}

type blockingGetter struct{}

func (blockingGetter) Handle(ctx context.Context, _ queries.GetPersonQuery) (ports.PersonView, error) {
	<-ctx.Done()
	return ports.PersonView{}, ctx.Err()
}

func TestRouter_RequestTimeoutReturnsJSONError(t *testing.T) {
	router := newTestRouter(t, HandlerDependencies{Get: blockingGetter{}}, func(options *RouterOptions) {
		options.ContextTimeout = 50 * time.Millisecond
	})

	resp := serve(router, http.MethodGet, "/person/"+uuid.New().String(), "", "")

	require.Contains(t, []int{http.StatusGatewayTimeout, http.StatusServiceUnavailable}, resp.Code, resp.Body.String())
	require.NotEmpty(t, decodeError(t, resp).Error.Code)
}

func TestRouter_ContextEnricherRunsForAPIRoutesOnly(t *testing.T) {
	var calls int
	router := newTestRouter(t, HandlerDependencies{Get: &fakeGetter{}}, func(options *RouterOptions) {
		options.ContextEnricher = func(ctx context.Context) context.Context {
			calls++
			return ctx
		}
	})

	serve(router, http.MethodGet, "/health/live", "", "")
	require.Equal(t, 0, calls)

	serve(router, http.MethodGet, "/person/"+uuid.New().String(), "", "")
	require.Equal(t, 1, calls)
}
