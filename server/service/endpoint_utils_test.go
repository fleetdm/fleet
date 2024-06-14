package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniversalDecoderIDs(t *testing.T) {
	type universalStruct struct {
		ID1        uint `url:"some-id"`
		OptionalID uint `url:"some-other-id,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, uint(0), casted.OptionalID)

	// fails if non optional IDs are not provided
	req = httptest.NewRequest("POST", "/target", nil)
	_, err = decoder(context.Background(), req)
	require.Error(t, err)
}

func TestUniversalDecoderIDsAndJSON(t *testing.T) {
	type universalStruct struct {
		ID1        uint   `url:"some-id"`
		SomeString string `json:"some_string"`
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "hello"}`
	req := httptest.NewRequest("POST", "/target", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, "hello", casted.SomeString)
}

func TestUniversalDecoderIDsAndJSONEmbedded(t *testing.T) {
	type EmbeddedJSON struct {
		SomeString string `json:"some_string"`
	}
	type UniversalStruct struct {
		ID1 uint `url:"some-id"`
		EmbeddedJSON
	}
	decoder := makeDecoder(UniversalStruct{})

	body := `{"some_string": "hello"}`
	req := httptest.NewRequest("POST", "/target", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*UniversalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, "hello", casted.SomeString)
}

func TestUniversalDecoderIDsAndListOptions(t *testing.T) {
	type universalStruct struct {
		ID1        uint              `url:"some-id"`
		Opts       fleet.ListOptions `url:"list_options"`
		SomeString string            `json:"some_string"`
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "bye"}`
	req := httptest.NewRequest("POST", "/target?per_page=77&page=4", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(123), casted.ID1)
	assert.Equal(t, "bye", casted.SomeString)
	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
}

func TestUniversalDecoderHandlersEmbeddedAndNot(t *testing.T) {
	type EmbeddedJSON struct {
		SomeString string `json:"some_string"`
	}
	type universalStruct struct {
		ID1  uint              `url:"some-id"`
		Opts fleet.ListOptions `url:"list_options"`
		EmbeddedJSON
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "o/"}`
	req := httptest.NewRequest("POST", "/target?per_page=77&page=4", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(123), casted.ID1)
	assert.Equal(t, "o/", casted.SomeString)
	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
}

func TestUniversalDecoderListOptions(t *testing.T) {
	type universalStruct struct {
		ID1  uint              `url:"some-id"`
		Opts fleet.ListOptions `url:"list_options"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	_, ok := decoded.(*universalStruct)
	require.True(t, ok)
}

func TestUniversalDecoderOptionalQueryParams(t *testing.T) {
	type universalStruct struct {
		ID1 *uint `query:"some_id,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Nil(t, casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_id=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	require.NotNil(t, casted.ID1)
	assert.Equal(t, uint(321), *casted.ID1)
}

func TestUniversalDecoderOptionalQueryParamString(t *testing.T) {
	type universalStruct struct {
		ID1 *string `query:"some_val,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Nil(t, casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_val=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	require.NotNil(t, casted.ID1)
	assert.Equal(t, "321", *casted.ID1)
}

func TestUniversalDecoderOptionalQueryParamNotPtr(t *testing.T) {
	type universalStruct struct {
		ID1 string `query:"some_val,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, "", casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_val=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, "321", casted.ID1)
}

func TestUniversalDecoderQueryAndListPlayNice(t *testing.T) {
	type universalStruct struct {
		ID1  *uint             `query:"some_id"`
		Opts fleet.ListOptions `url:"list_options"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target?per_page=77&page=4&some_id=444", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
	require.NotNil(t, casted.ID1)
	assert.Equal(t, uint(444), *casted.ID1)
}

type stringErrorer string

func (s stringErrorer) error() error { return nil }

func TestEndpointer(t *testing.T) {
	r := mux.NewRouter()
	ds := new(mock.Store)
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			ID:         3,
			UserID:     42,
			Key:        key,
			AccessedAt: time.Now(),
		}, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return []*fleet.User{{GlobalRole: ptr.String(fleet.RoleAdmin)}}, nil
	}

	svc, _ := newTestService(t, ds, nil, nil)

	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext, // populate the request context with common fields
			setRequestsContexts(svc),
		),
		kithttp.ServerErrorHandler(&errorHandler{kitlog.NewNopLogger()}),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			logRequestEnd(kitlog.NewNopLogger()),
			checkLicenseExpiration(svc),
		),
	}

	e := newUserAuthenticatedEndpointer(svc, fleetAPIOptions, r, "v1", "2021-11")
	nopHandler := func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
		setAuthCheckedOnPreAuthErr(ctx)
		return stringErrorer("nop"), nil
	}
	overrideHandler := func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
		setAuthCheckedOnPreAuthErr(ctx)
		return stringErrorer("override"), nil
	}

	// Regular path, no plan to deprecate
	e.GET("/api/_version_/fleet/path1", nopHandler, struct{}{})

	// New path, we want it only available starting from the specified version
	e.StartingAtVersion("2021-11").GET("/api/_version_/fleet/newpath", nopHandler, struct{}{})

	// Path that was in v1, but was changed in 2021-11
	e.EndingAtVersion("v1").GET("/api/_version_/fleet/overriddenpath", nopHandler, struct{}{})
	e.StartingAtVersion("2021-11").GET("/api/_version_/fleet/overriddenpath", overrideHandler, struct{}{})

	// Path that got deprecated
	e.EndingAtVersion("v1").GET("/api/_version_/fleet/deprecated", nopHandler, struct{}{})
	// Path that got deprecated but in the latest version
	e.EndingAtVersion("2021-11").GET("/api/_version_/fleet/deprecated-soon", nopHandler, struct{}{})

	// Aliasing works with versioning too
	e.WithAltPaths("/api/_version_/fleet/something/{fff}").GET("/api/_version_/fleet/somethings/{fff}", nopHandler, struct{}{})

	mustMatch := []struct {
		method     string
		path       string
		overridden bool
	}{
		{method: "GET", path: "/api/v1/fleet/path1"},
		{method: "GET", path: "/api/2021-11/fleet/path1"},
		{method: "GET", path: "/api/latest/fleet/path1"},

		{method: "GET", path: "/api/2021-11/fleet/newpath"},
		{method: "GET", path: "/api/latest/fleet/newpath"},

		{method: "GET", path: "/api/v1/fleet/deprecated"},

		{method: "GET", path: "/api/v1/fleet/deprecated-soon"},
		{method: "GET", path: "/api/2021-11/fleet/deprecated-soon"},
		{method: "GET", path: "/api/latest/fleet/deprecated-soon"},

		{method: "GET", path: "/api/v1/fleet/overriddenpath"},
		{method: "GET", path: "/api/2021-11/fleet/overriddenpath", overridden: true},
		{method: "GET", path: "/api/latest/fleet/overriddenpath", overridden: true},

		{method: "GET", path: "/api/v1/fleet/something/aaa"},
		{method: "GET", path: "/api/2021-11/fleet/something/aaa"},
		{method: "GET", path: "/api/latest/fleet/something/aaa"},
		{method: "GET", path: "/api/v1/fleet/somethings/aaa"},
		{method: "GET", path: "/api/2021-11/fleet/somethings/aaa"},
		{method: "GET", path: "/api/latest/fleet/somethings/aaa"},
	}

	mustNotMatch := []struct {
		method  string
		path    string
		handler http.Handler
	}{
		{method: "POST", path: "/api/v1/fleet/path1"},
		{method: "GET", path: "/api/v1/fleet/qwejoqiwejqiowehioqwe"},
		{method: "GET", path: "/api/v1/qwejoqiwejqiowehioqwe"},

		{method: "GET", path: "/api/v1/fleet/newpath"},

		{method: "GET", path: "/api/2021-11/fleet/deprecated"},
		{method: "GET", path: "/api/latest/fleet/deprecated"},
	}

	doesItMatch := func(method, path string, override bool) bool {
		testURL := url.URL{Path: path}
		request := http.Request{Method: method, URL: &testURL, Header: map[string][]string{"Authorization": {"Bearer asd"}}, Body: io.NopCloser(strings.NewReader(""))}
		routeMatch := mux.RouteMatch{}

		res := r.Match(&request, &routeMatch)
		if routeMatch.Route != nil {
			rec := httptest.NewRecorder()
			routeMatch.Handler.ServeHTTP(rec, &request)
			got := rec.Body.String()
			if override {
				require.Equal(t, "\"override\"\n", got)
			} else {
				require.Equal(t, "\"nop\"\n", got)
			}
		}
		return res && routeMatch.MatchErr == nil && routeMatch.Route != nil
	}

	for _, route := range mustMatch {
		require.True(t, doesItMatch(route.method, route.path, route.overridden), route)
	}

	for _, route := range mustNotMatch {
		require.False(t, doesItMatch(route.method, route.path, false), route)
	}
}

func TestEndpointerCustomMiddleware(t *testing.T) {
	r := mux.NewRouter()
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)

	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext,
			setRequestsContexts(svc),
		),
		kithttp.ServerErrorHandler(&errorHandler{kitlog.NewNopLogger()}),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			logRequestEnd(kitlog.NewNopLogger()),
			checkLicenseExpiration(svc),
		),
	}

	var buf bytes.Buffer
	e := newNoAuthEndpointer(svc, fleetAPIOptions, r, "v1")
	e.GET("/none/", func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
		buf.WriteString("H1")
		return nil, nil
	}, nil)

	e.WithCustomMiddleware(
		func(e endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (response interface{}, err error) {
				buf.WriteString("A")
				return e(ctx, request)
			}
		},
		func(e endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (response interface{}, err error) {
				buf.WriteString("B")
				return e(ctx, request)
			}
		},
		func(e endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (response interface{}, err error) {
				buf.WriteString("C")
				return e(ctx, request)
			}
		},
	).
		GET("/mw/", func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
			buf.WriteString("H2")
			return nil, nil
		}, nil)

	req := httptest.NewRequest("GET", "/none/", nil)
	var m1 mux.RouteMatch

	require.True(t, r.Match(req, &m1))
	rec := httptest.NewRecorder()
	m1.Handler.ServeHTTP(rec, req)
	require.Equal(t, "H1", buf.String())

	buf.Reset()
	req = httptest.NewRequest("GET", "/mw/", nil)
	var m2 mux.RouteMatch

	require.True(t, r.Match(req, &m2))
	rec = httptest.NewRecorder()
	m2.Handler.ServeHTTP(rec, req)
	require.Equal(t, "ABCH2", buf.String())
}

func TestWriteBrowserSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	writeBrowserSecurityHeaders(w)
	headers := w.Header()
	require.Equal(
		t,
		http.Header{
			"X-Content-Type-Options":    {"nosniff"},
			"X-Frame-Options":           {"SAMEORIGIN"},
			"Strict-Transport-Security": {"max-age=31536000; includeSubDomains;"},
			"Referrer-Policy":           {"strict-origin-when-cross-origin"},
		},
		headers,
	)
}
