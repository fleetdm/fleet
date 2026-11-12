package jsondecode

import (
	jsonv1 "encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// passthrough mirrors the pkg/optjson scalars: it hands an encoding/json error straight back, which is
// the case that lost its position in Go 1.27 and the reason this package exists.
type passthrough struct{ V int }

func (p *passthrough) UnmarshalJSON(b []byte) error { return jsonv1.Unmarshal(b, &p.V) }

// composite mirrors optjson.Slice[T]: a custom unmarshaler over a list rather than a scalar.
type composite struct{ V []passthrough }

func (c *composite) UnmarshalJSON(b []byte) error { return jsonv1.Unmarshal(b, &c.V) }

// errNotBase64 stands in for a domain validation failure, as server/service's
// backwardsCompatProfilesParam raises when profile contents will not decode.
var errNotBase64 = errors.New("contents must be base64")

type validated struct{ V string }

func (s *validated) UnmarshalJSON(b []byte) error {
	var raw string
	if err := jsonv1.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw != "ok" {
		return fmt.Errorf("field %q: %w", raw, errNotBase64)
	}
	return nil
}

// wrapping adds context to an encoding/json failure instead of returning it untouched.
type wrapping struct{ V map[string][]byte }

func (w *wrapping) UnmarshalJSON(b []byte) error {
	if err := jsonv1.Unmarshal(b, &w.V); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using old format: %w", err)
	}
	return nil
}

// port uses strconv, so the error it returns looks exactly like the one v2 raises for a number that
// will not fit a Go numeric type.
type port struct{ V int }

func (p *port) UnmarshalJSON(b []byte) error {
	var raw string
	if err := jsonv1.Unmarshal(b, &raw); err != nil {
		return err
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("port must be numeric: %w", err)
	}
	p.V = v
	return nil
}

type payload struct {
	Plain     string      `json:"plain"`
	Num       int         `json:"num"`
	Nums      []int       `json:"nums"`
	Custom    passthrough `json:"custom"`
	List      composite   `json:"list"`
	Validated validated   `json:"validated"`
	Wrapping  wrapping    `json:"wrapping"`
	Port      port        `json:"port"`
	Items     []payload   `json:"items"`
}

// TestMatchesEncodingJSON is the main guarantee: for everything except the errors that lost their
// position (see TestAddsFieldPathWhereEncodingJSONCannot), a caller must see exactly the error
// encoding/json would have produced. Comparing against the real decoder rather than a hard-coded string
// keeps this honest across Go releases.
func TestMatchesEncodingJSON(t *testing.T) {
	for _, in := range []string{
		// Plain type mismatches, where v2 reports no underlying cause.
		`{"plain":123}`,
		`{"num":"x"}`,
		`{"nums":true}`,
		`{"nums":[1,"two"]}`,
		// Numbers that will not fit, where strconv is the cause but the result is still a type error.
		`{"num":1.23}`,
		`{"num":1e400}`,
		// Errors raised by a custom UnmarshalJSON, which must survive verbatim.
		`{"validated":"nope"}`,
		`{"wrapping":{"a":123}}`,
		`{"wrapping":123}`,
		`{"port":"http"}`,
	} {
		t.Run(in, func(t *testing.T) {
			var want, got payload
			wantErr := jsonv1.Unmarshal([]byte(in), &want)
			gotErr := Unmarshal([]byte(in), &got)

			require.Error(t, wantErr, "fixture should fail to decode")
			require.EqualError(t, gotErr, wantErr.Error())
			require.Equal(t, IsTypeError(wantErr), IsTypeError(gotErr))
		})
	}
}

// TestAddsFieldPathWhereEncodingJSONCannot covers the regression this package fixes. A custom
// UnmarshalJSON that returns an encoding/json error reaches the caller stripped of its position under
// Go 1.27, so here alone the error is deliberately *not* identical to encoding/json's.
func TestAddsFieldPathWhereEncodingJSONCannot(t *testing.T) {
	for _, c := range []struct {
		in       string
		wantPath string
	}{
		{`{"custom":"nope"}`, "custom"},
		{`{"items":[{"custom":"nope"}]}`, "items.0.custom"},
		{`{"items":[{"custom":1},{"custom":"nope"}]}`, "items.1.custom"},
	} {
		t.Run(c.in, func(t *testing.T) {
			var bare payload
			bareErr := jsonv1.Unmarshal([]byte(c.in), &bare)
			bareType, ok := errors.AsType[*jsonv1.UnmarshalTypeError](bareErr)
			require.True(t, ok)
			require.Empty(t, bareType.Field, "encoding/json cannot locate this on its own")

			err := Unmarshal([]byte(c.in), &payload{})
			terr, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err)
			require.True(t, ok, "callers type-switch on *json.UnmarshalTypeError")
			require.Equal(t, c.wantPath, terr.Field)
			require.Equal(t, "payload", terr.Struct)
			// The concrete type that failed, not the wrapper around it.
			require.Equal(t, "int", terr.Type.String())
			require.Equal(t, "string", terr.Value)
		})
	}
}

// TestStopsAtCompositeUnmarshalerBoundary documents a known limit. Locating a failure relies on v2 error
// semantics, and v2 cannot see inside a custom UnmarshalJSON: it reports the position of the value
// handed to the method, not of the field within it. For a scalar wrapper that is the whole path, but for
// a composite the path stops at the list.
func TestStopsAtCompositeUnmarshalerBoundary(t *testing.T) {
	err := Unmarshal([]byte(`{"list":[{"v":"nope"}]}`), &payload{})
	require.Equal(t, "list", FieldPath(err))
}

// TestValidationErrorsAreNotTypeErrors guards the distinction that makes this package safe to put in
// front of a decode: a wrong type is reported as a type error with the path filled in, but a validation
// failure is the caller's own description of what is wrong and must not be rewritten into one.
func TestValidationErrorsAreNotTypeErrors(t *testing.T) {
	t.Run("plain validation error", func(t *testing.T) {
		err := Unmarshal([]byte(`{"validated":"nope"}`), &payload{})
		require.ErrorIs(t, err, errNotBase64, "the sentinel must stay reachable")
		require.False(t, IsTypeError(err))
	})

	t.Run("strconv error from a custom unmarshaler", func(t *testing.T) {
		// The number-overflow branch keys off a strconv error, so it also checks the target type;
		// otherwise this would be rewritten into "cannot unmarshal string into ...".
		err := Unmarshal([]byte(`{"port":"http"}`), &payload{})
		require.ErrorIs(t, err, strconv.ErrSyntax)
		require.False(t, IsTypeError(err))
	})
}

func TestMalformedJSONKeepsV1Wording(t *testing.T) {
	for _, in := range []string{`{`, ``, `{"plain":`} {
		var want, got payload
		err := Unmarshal([]byte(in), &got)
		require.EqualError(t, err, jsonv1.Unmarshal([]byte(in), &want).Error(), in)
		// The endpointer tells a truncated body apart from an exhausted size limit by this sentinel, so
		// restating the message must not break the chain.
		require.ErrorIs(t, err, io.ErrUnexpectedEOF, in)
	}

	// Other syntax errors keep encoding/json's leading wording, without jsontext's prefix.
	err := Unmarshal([]byte(`{"plain":}`), &payload{})
	require.ErrorContains(t, err, "invalid character '}'")
	require.NotContains(t, err.Error(), "jsontext")
	require.False(t, IsTypeError(err))
}

func TestUnknownFieldKeepsV1Wording(t *testing.T) {
	// Some code detects this case by matching encoding/json's exact message, so it has to survive
	// verbatim, and only when the caller asked for strictness.
	err := NewDecoder(strings.NewReader(`{"nope":1}`), RejectUnknownMembers()).Decode(&payload{})
	require.EqualError(t, err, `json: unknown field "nope"`)

	require.NoError(t, Unmarshal([]byte(`{"nope":1}`), &payload{}))
}

func TestPreservesEncodingJSONDecodingBehavior(t *testing.T) {
	t.Run("member matching stays case-insensitive", func(t *testing.T) {
		var v payload
		require.NoError(t, Unmarshal([]byte(`{"PLAIN":"x"}`), &v))
		require.Equal(t, "x", v.Plain)
	})

	t.Run("duplicate names allowed", func(t *testing.T) {
		require.NoError(t, Unmarshal([]byte(`{"plain":"a","plain":"b"}`), &payload{}))
	})

	t.Run("Decode leaves trailing bytes, Unmarshal rejects them", func(t *testing.T) {
		// JSONStrictDecode reads a second value to detect trailing content, and relies on io.EOF here.
		dec := NewDecoder(strings.NewReader(`{"plain":"a"}{"plain":"b"}`))

		var first, second payload
		require.NoError(t, dec.Decode(&first))
		require.Equal(t, "a", first.Plain)
		require.NoError(t, dec.Decode(&second))
		require.Equal(t, "b", second.Plain)
		require.ErrorIs(t, dec.Decode(&second), io.EOF)

		require.Error(t, Unmarshal([]byte(`{"plain":"a"}{"plain":"b"}`), &payload{}))
	})
}

func TestIsTypeErrorAndFieldPathOnForeignErrors(t *testing.T) {
	// Both are used on errors this package did not produce, so they have to cope with anything.
	var v payload
	require.True(t, IsTypeError(jsonv1.Unmarshal([]byte(`{"plain":123}`), &v)))
	require.Equal(t, "a.b", FieldPath(&jsonv1.UnmarshalTypeError{Field: "a.b"}))

	for _, err := range []error{errors.New("boom"), nil} {
		require.False(t, IsTypeError(err))
		require.Empty(t, FieldPath(err))
	}
}
