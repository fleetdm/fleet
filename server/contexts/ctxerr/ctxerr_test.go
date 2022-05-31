package ctxerr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	pkgerrors "github.com/pkg/errors" //nolint:depguard
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHandler struct {
	StoreImpl func(err error)
}

func (h mockHandler) Store(err error) {
	h.StoreImpl(err)
}

func setup() (context.Context, func()) {
	ctx := context.Background()
	eh := mockHandler{}
	ctx = NewContext(ctx, eh)
	nowFn = func() time.Time {
		now, _ := time.Parse(time.RFC3339, "1969-06-19T21:44:05Z")
		return now
	}

	return ctx, func() { nowFn = time.Now }
}

func TestCause(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()

	errNew := errors.New("new")
	fmtWrap := fmt.Errorf("fmt: %w", errNew)
	pkgWrap := pkgerrors.Wrap(errNew, "pkg")
	pkgFmtWrap := pkgerrors.Wrap(fmtWrap, "pkg")
	fmtPkgWrap := fmt.Errorf("fmt: %w", pkgWrap)
	ctxNew := New(ctx, "ctxerr")
	ctxWrap := Wrap(ctx, ctxNew, "wrap")
	ctxDoubleWrap := Wrap(ctx, ctxWrap, "re-wrap")
	pkgFmtCtxWrap := pkgerrors.Wrap(fmt.Errorf("fmt: %w", ctxWrap), "pkg")
	fmtPkgCtxWrap := fmt.Errorf("fmt: %w", pkgerrors.Wrap(ctxWrap, "pkg"))

	cases := []struct {
		in, out error
	}{
		{nil, nil},
		{io.EOF, io.EOF},
		{errNew, errNew},
		{fmtWrap, errNew},
		{pkgWrap, errNew},
		{pkgFmtWrap, errNew},
		{fmtPkgWrap, errNew},
		{ctxNew, ctxNew},
		{ctxWrap, ctxNew},
		{ctxDoubleWrap, ctxNew},
		{pkgFmtCtxWrap, ctxNew},
		{fmtPkgCtxWrap, ctxNew},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%T: %[1]v", c.in), func(t *testing.T) {
			got := Cause(c.in)
			require.Equal(t, c.out, got)
		})
	}
}

func TestNew(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	err := New(ctx, "new").(*FleetError)

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestNewWithData(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		data := map[string]interface{}{"foo": "bar"}
		err := NewWithData(ctx, "new", data).(*FleetError)

		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.Nil(t, err.cause)
		require.Equal(t, err.data, json.RawMessage(`{"foo":"bar","timestamp":"1969-06-19T21:44:05Z"}`))
	})

	t.Run("with invalid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		data := map[string]interface{}{"foo": make(chan int)}
		err := NewWithData(ctx, "new", data).(*FleetError)
		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.Nil(t, err.cause)
		assert.Regexp(t, regexp.MustCompile(`{"error": ".+"}`), string(err.data))
	})
}

func TestErrorf(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	err := Errorf(ctx, "%s %d", "new", 1).(*FleetError)

	require.Equal(t, err.msg, "new 1")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestWrap(t *testing.T) {
	t.Run("with message provided", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		cause := errors.New("cause")
		err := Wrap(ctx, cause, "new").(*FleetError)

		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
	})

	t.Run("without message provided", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		cause := errors.New("cause")
		err := Wrap(ctx, cause).(*FleetError)
		require.Equal(t, err.msg, "")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
	})

	t.Run("with nil error provided", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		err := Wrap(ctx, nil)
		require.Equal(t, err, nil)
	})
}

func TestWrapNewWithData(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		cause := errors.New("cause")
		data := map[string]interface{}{"foo": "bar"}
		err := WrapWithData(ctx, cause, "new", data).(*FleetError)

		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
		require.Equal(t, err.data, json.RawMessage(`{"foo":"bar","timestamp":"1969-06-19T21:44:05Z"}`))
	})

	t.Run("with invalid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		cause := errors.New("cause")
		data := map[string]interface{}{"foo": make(chan int)}
		err := WrapWithData(ctx, cause, "new", data).(*FleetError)
		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
		assert.Regexp(t, regexp.MustCompile(`{"error": ".+"}`), string(err.data))
	})

	t.Run("without message provided", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		data := map[string]interface{}{"foo": make(chan int)}
		cause := errors.New("cause")
		err := WrapWithData(ctx, cause, "", data).(*FleetError)
		require.Equal(t, err.msg, "")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
		assert.Regexp(t, regexp.MustCompile(`{"error": ".+"}`), string(err.data))
	})

	t.Run("with nil error provided", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		err := WrapWithData(ctx, nil, "msg", map[string]interface{}{"foo": "bar"})
		require.Equal(t, err, nil)
	})
}

func TestWrapf(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	cause := errors.New("cause")
	err := Wrapf(ctx, cause, "%s %d", "new", 1).(*FleetError)

	require.Equal(t, err.msg, "new 1")
	require.NotEmpty(t, err.stack.List())
	require.NotNil(t, err.cause)
}

func TestUnwrap(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	cause := errors.New("cause")
	err := Wrap(ctx, cause, "new")

	require.Equal(t, Unwrap(err), cause)
}

func TestFleetErrorMarshalling(t *testing.T) {
	cases := []struct {
		msg string
		in  FleetError
		out string
	}{
		{"only error", FleetError{"a", mockStack{}, nil, nil}, `{"message": "a"}`},
		{"errors and stack", FleetError{"a", mockStack{[]string{"test"}}, errors.New("err"), nil}, `{"message": "a", "stack": ["test"]}`},
		{
			"errors, stack and data",
			FleetError{"a", mockStack{[]string{"test"}}, errors.New("err"), json.RawMessage(`{"foo":"bar"}`)},
			`{"message": "a", "stack": ["test"], "data": {"foo": "bar"}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			json, err := c.in.MarshalJSON()

			require.NoError(t, err)
			require.JSONEq(t, c.out, string(json))
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()

	errNew := errors.New("a")

	errWrap := Wrap(ctx, errNew, "b").(*FleetError)
	errWrap.stack = mockStack{[]string{"sb"}}

	errNewWithData := NewWithData(ctx, "c", map[string]interface{}{"f": "c"}).(*FleetError)
	errNewWithData.stack = mockStack{[]string{"sc"}}

	cases := []struct {
		msg string
		in  error
		out string
	}{
		{
			"non-wrapped errors",
			errNew,
			`{"cause": {"message": "a"}}`,
		},
		{
			"wrapped error",
			errWrap,
			`{"cause": {"message": "a"}, "wraps": [{"message": "b", "data": {"timestamp": "1969-06-19T21:44:05Z"}, "stack": ["sb"]}]}`,
		},
		{
			"wrapped error with data",
			errNewWithData,
			`{"cause": {"message": "c", "stack": ["sc"], "data": {"f": "c", "timestamp": "1969-06-19T21:44:05Z"}}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			json, err := MarshalJSON(c.in)
			require.NoError(t, err)
			require.JSONEq(t, c.out, string(json))
		})
	}
}

func TestStackMethod(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()

	errNew := errors.New("a")
	errWrap := Wrap(ctx, errNew, "b").(*FleetError)
	errWrap.stack = mockStack{[]string{"sb"}}

	require.Equal(t, []string{"sb"}, errWrap.Stack())
}

func TestFleetCause(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()

	var nilErr *FleetError
	errNew := errors.New("a")
	errWrapRoot := Wrap(ctx, errNew, "wrapRoot")
	errWrap1 := Wrap(ctx, errWrapRoot, "wrap1")
	errWrap2 := Wrap(ctx, errWrap1, "wrap2")

	cases := []struct {
		msg string
		in  error
		out error
	}{
		{"non-fleet, unwrapped errors returns nil", errNew, nilErr},
		{"fleet unwrapped errors returns the error itself", errWrapRoot, errWrapRoot},
		{"deeply nested errors return the root fleet error", errWrap1, errWrapRoot},
		{"deeply nested errors return the root fleet error", errWrap2, errWrapRoot},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			actual := FleetCause(c.in)
			require.Equal(t, c.out, actual)
		})
	}
}

func TestHandle(t *testing.T) {
	t.Run("stores the error when invoked", func(t *testing.T) {
		ctx := context.Background()
		eh := mockHandler{}
		err := New(ctx, "new")
		eh.StoreImpl = func(serr error) {
			require.Equal(t, serr, err)
		}
		ctx = NewContext(ctx, eh)
		Handle(ctx, err)
	})

	t.Run("wraps when there's no FleetError in the chain", func(t *testing.T) {
		ctx := context.Background()
		eh := mockHandler{}
		err := errors.New("new")
		eh.StoreImpl = func(serr error) {
			var ferr *FleetError
			require.ErrorAs(t, serr, &ferr)
		}
		ctx = NewContext(ctx, eh)
		Handle(ctx, err)
	})
}

func TestAdditionalMetadata(t *testing.T) {
	t.Run("saves additional data about the host if present", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		hctx := host.NewContext(ctx, &fleet.Host{Platform: "test_platform", OsqueryVersion: "5.0"})
		err := New(hctx, "with host context").(*FleetError)

		require.JSONEq(t, string(err.data), `{"host":{"osquery_version":"5.0","platform":"test_platform"},"timestamp":"1969-06-19T21:44:05Z"}`)
	})

	t.Run("saves additional data about the viewer if present", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		vctx := viewer.NewContext(ctx, viewer.Viewer{Session: &fleet.Session{ID: 1}, User: &fleet.User{SSOEnabled: true}})
		err := New(vctx, "with host context").(*FleetError)

		require.JSONEq(t, string(err.data), `{"viewer":{"is_logged_in":true,"sso_enabled":true},"timestamp":"1969-06-19T21:44:05Z"}`)
	})
}
