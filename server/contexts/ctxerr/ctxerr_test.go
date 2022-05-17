package ctxerr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors" //nolint:depguard
	"github.com/stretchr/testify/require"
)

type mockHandler struct{}

func (h mockHandler) Store(err error) {}

func setup() context.Context {
	ctx := context.Background()
	eh := mockHandler{}
	ctx = NewContext(ctx, eh)
	return ctx
}

func TestCause(t *testing.T) {
	ctx := setup()

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
	ctx := setup()
	err := New(ctx, "new")

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestNewWithData(t *testing.T) {
	ctx := setup()
	data := map[string]interface{}{"foo": "bar"}
	err := NewWithData(ctx, "new", data)

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
	require.Equal(t, err.data, data)
}

func TestErrorf(t *testing.T) {
	ctx := setup()
	err := Errorf(ctx, "%s %d", "new", 1)

	require.Equal(t, err.msg, "new 1")
	require.NotEmpty(t, err.stack.List())
	require.Nil(t, err.cause)
}

func TestWrap(t *testing.T) {
	ctx := setup()
	cause := errors.New("cause")
	err := Wrap(ctx, cause, "new")

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.NotNil(t, err.cause)
}

func TestWrapNewWithData(t *testing.T) {
	ctx := setup()
	cause := errors.New("cause")
	data := map[string]interface{}{"foo": "bar"}
	err := WrapWithData(ctx, cause, "new", data)

	require.Equal(t, err.msg, "new")
	require.NotEmpty(t, err.stack.List())
	require.NotNil(t, err.cause)
	require.Equal(t, err.data, data)
}

func TestWrapf(t *testing.T) {
	ctx := setup()
	cause := errors.New("cause")
	err := Wrapf(ctx, cause, "%s %d", "new", 1)

	require.Equal(t, err.msg, "new 1")
	require.NotEmpty(t, err.stack.List())
	require.NotNil(t, err.cause)
}

func TestUnwrap(t *testing.T) {
	ctx := setup()
	cause := errors.New("cause")
	err := Wrap(ctx, cause, "new")

	require.Equal(t, Unwrap(err), cause)
}

func TestUnpack(t *testing.T) {
	ctx := setup()
	cause := errors.New("cause")
	err := Wrap(ctx, cause, "new")
	err = Wrap(ctx, err, "nested")
	err = Wrap(ctx, err, "nested 2")

	scause, stack := Summarize(err)

	require.Equal(t, scause, cause)
	require.NotEmpty(t, stack)
}

func TestFleetErrorMarshalling(t *testing.T) {
	cases := []struct {
		msg string
		in  FleetError
		out string
	}{
		{"only error", FleetError{"a", mockStack{}, nil, nil}, `{"Message": "a"}`},
		{"errors and stack", FleetError{"a", mockStack{[]string{"test"}}, errors.New("err"), nil}, `{"Message": "a", "Stack": ["test"]}`},
		{
			"errors, stack and data",
			FleetError{"a", mockStack{[]string{"test"}}, errors.New("err"), map[string]interface{}{"foo": "bar"}},
			`{"Message": "a", "Stack": ["test"], "Data": {"foo": "bar"}}`,
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
	nowFn = func() time.Time {
		now, _ := time.Parse(time.RFC3339, "1969-06-19T21:44:05Z")
		return now
	}
	defer func() { nowFn = time.Now }()
	ctx := setup()

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
			`{"Cause": {"Message": "a"}, "Wraps": []}`,
		},
		{
			"wrapped error",
			errWrap,
			`{"Cause": {"Message": "a"}, "Wraps": [{"Data": {"Timestamp": "1969-06-19T21:44:05Z"}, "Stack": ["sa"]}]}`,
		},
		{
			"wrapped error with data",
			errNewWithData,
			`{"Cause": {"Message": "b", "Stack": ["sb"], "Data": {"f": "b", "Timestamp": "1969-06-19T21:44:05Z"}}, "Wraps": []}`,
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
