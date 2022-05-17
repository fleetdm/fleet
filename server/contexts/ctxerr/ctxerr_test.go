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

	pkgerrors "github.com/pkg/errors" //nolint:depguard
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type mockHandler struct {
	StoreImpl func(err error)
}

func (h mockHandler) Store(err error) {
	h.StoreImpl(err)
}

type cleanupFn func()

func setup() (context.Context, cleanupFn) {
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
	err := New(ctx, "new")

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestNewWithData(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		data := map[string]interface{}{"foo": "bar"}
		err := NewWithData(ctx, "new", data)

		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.Nil(t, err.cause)
		require.Equal(t, err.data, json.RawMessage(`{"foo":"bar","timestamp":"1969-06-19T21:44:05Z"}`))
	})

	t.Run("with invalid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		data := map[string]interface{}{"foo": make(chan int)}
		err := NewWithData(ctx, "new", data)
		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.Nil(t, err.cause)
		assert.Regexp(t, regexp.MustCompile(`{"error": ".+"}`), string(err.data))
	})
}

func TestErrorf(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	err := Errorf(ctx, "%s %d", "new", 1)

	require.Equal(t, err.msg, "new 1")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestWrap(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	cause := errors.New("cause")
	err := Wrap(ctx, cause, "new")

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.NotNil(t, err.cause)
}

func TestWrapNewWithData(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		ctx, cleanup := setup()
		defer cleanup()
		cause := errors.New("cause")
		data := map[string]interface{}{"foo": "bar"}
		err := WrapWithData(ctx, cause, "new", data)

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
		err := WrapWithData(ctx, cause, "new", data)
		require.Equal(t, err.msg, "new")
		require.NotEmpty(t, err.stack.List())
		require.NotNil(t, err.cause)
		assert.Regexp(t, regexp.MustCompile(`{"error": ".+"}`), string(err.data))
	})
}

func TestWrapf(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()
	cause := errors.New("cause")
	err := Wrapf(ctx, cause, "%s %d", "new", 1)

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

	errWrap := Wrap(ctx, errNew)
	errWrap.stack = mockStack{[]string{"sa"}}

	errNewWithData := NewWithData(ctx, "b", map[string]interface{}{"f": "b"})
	errNewWithData.stack = mockStack{[]string{"sb"}}

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
			`{"cause": {"message": "a"}, "wraps": [{"data": {"timestamp": "1969-06-19T21:44:05Z"}, "stack": ["sa"]}]}`,
		},
		{
			"wrapped error with data",
			errNewWithData,
			`{"cause": {"message": "b", "stack": ["sb"], "data": {"f": "b", "timestamp": "1969-06-19T21:44:05Z"}}}`,
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
	errWrap := Wrap(ctx, errNew)
	errWrap.stack = mockStack{[]string{"sa"}}

	require.Equal(t, []string{"sa"}, errWrap.Stack())
}

func TestFleetCause(t *testing.T) {
	ctx, cleanup := setup()
	defer cleanup()

	var nilErr *FleetError = nil
	errNew := errors.New("a")
	errWrapRoot := Wrap(ctx, errNew)
	errWrap1 := Wrap(ctx, errWrapRoot)
	errWrap2 := Wrap(ctx, errWrap1)

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
			fmt.Println(c.msg)
			fmt.Println(c.out)
			fmt.Println(actual)
			require.Equal(t, c.out, actual)
		})
	}
}

func TestHandle(t *testing.T) {
	ctx := context.Background()
	eh := mockHandler{}
	err := New(ctx, "new")
	eh.StoreImpl = func(serr error) {
		require.Equal(t, serr, err)
	}
	ctx = NewContext(ctx, eh)
	Handle(ctx, err)
}
