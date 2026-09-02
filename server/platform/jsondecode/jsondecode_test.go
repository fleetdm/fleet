package jsondecode

import (
	jsonv1 "encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// optInt stands in for the pkg/optjson types: a custom UnmarshalJSON that delegates to encoding/json.
// These are the types whose errors lost all position information in Go 1.27.
type optInt struct {
	Set, Valid bool
	Value      int
}

func (o *optInt) UnmarshalJSON(b []byte) error {
	o.Set = true
	if err := jsonv1.Unmarshal(b, &o.Value); err != nil {
		return err
	}
	o.Valid = true
	return nil
}

type inner struct {
	Plain  string `json:"plain"`
	Custom optInt `json:"custom"`
	Nums   []int  `json:"nums"`
}

type root struct {
	Inner inner   `json:"inner"`
	Specs []inner `json:"specs"`
}

// decode reproduces what a caller does today: decode with encoding/json, then enrich on failure.
func decode(t *testing.T, in string) (*jsonv1.UnmarshalTypeError, error) {
	t.Helper()
	var v root
	err := Enrich(jsonv1.Unmarshal([]byte(in), &v), []byte(in), &v)
	terr, _ := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	return terr, err
}

func TestEnrichRecoversPathLostByCustomUnmarshaler(t *testing.T) {
	for _, c := range []struct {
		name     string
		in       string
		wantPath string
	}{
		{"custom unmarshaler", `{"inner":{"custom":"nope"}}`, "inner.custom"},
		{"through a slice of structs", `{"specs":[{"custom":"nope"}]}`, "specs.0.custom"},
		{"second element", `{"specs":[{"custom":1},{"custom":"nope"}]}`, "specs.1.custom"},
	} {
		t.Run(c.name, func(t *testing.T) {
			terr, err := decode(t, c.in)
			require.Error(t, err)
			require.NotNil(t, terr, "callers type-switch on *json.UnmarshalTypeError")

			require.Equal(t, c.wantPath, terr.Field)
			require.Equal(t, "root", terr.Struct)
			// The concrete type that failed, not the optjson wrapper around it.
			require.Equal(t, "int", terr.Type.String())
			require.Equal(t, "string", terr.Value)
		})
	}
}

func TestEnrichLeavesAlreadyLocatedErrorsAlone(t *testing.T) {
	// Plain fields never lost their path, so Enrich must not touch them: re-deriving it would risk
	// disagreeing with what encoding/json already reported.
	for _, c := range []struct {
		name     string
		in       string
		wantPath string
	}{
		{"plain field", `{"inner":{"plain":123}}`, "inner.plain"},
		{"slice element", `{"inner":{"nums":[1,"two"]}}`, "inner.nums"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var v root
			before := jsonv1.Unmarshal([]byte(c.in), &v)
			var beforeType *jsonv1.UnmarshalTypeError
			require.ErrorAs(t, before, &beforeType)
			original := *beforeType

			after, _ := decode(t, c.in)
			require.NotNil(t, after)
			require.Equal(t, original.Field, after.Field)
			require.Equal(t, original.Type, after.Type)
			require.Equal(t, original.Value, after.Value)
			require.Contains(t, after.Field, c.wantPath)
		})
	}
}

// optSlice mirrors optjson.Slice[T]: a custom unmarshaler over a *composite*, which delegates the whole
// element list to encoding/json.
type optSlice[T any] struct {
	Set   bool
	Value []T
}

func (s *optSlice[T]) UnmarshalJSON(b []byte) error {
	s.Set = true
	return jsonv1.Unmarshal(b, &s.Value)
}

// TestEnrichStopsAtCompositeUnmarshalerBoundary documents a known limit. Locating a failure relies on
// v2 error semantics, and v2 cannot see inside a custom UnmarshalJSON: it reports the position of the
// value handed to the method, not of the field within it. For a scalar wrapper such as optjson.Int that
// is the whole path, but for a composite such as optjson.Slice the path stops at the slice.
//
// Go 1.26 reported the full "specs.software.packages.self_service" here, because the v1 decoder
// accumulated its field stack across the custom unmarshaler. Recovering that last segment again would
// mean giving the optjson composites a v2 UnmarshalJSONFrom method, which changes how they decode
// everywhere v2 is used, so it is deliberately out of scope here. A truncated-but-correct prefix is
// still a large improvement on the empty path this package exists to fix.
func TestEnrichStopsAtCompositeUnmarshalerBoundary(t *testing.T) {
	type pkg struct {
		SelfService optInt `json:"self_service"`
	}
	type software struct {
		Packages optSlice[pkg] `json:"packages"`
	}
	type spec struct {
		Software software `json:"software"`
	}
	type request struct {
		Specs []spec `json:"specs"`
	}

	in := []byte(`{"specs":[{"software":{"packages":[{"self_service":"yes"}]}}]}`)
	var v request
	err := Enrich(jsonv1.Unmarshal(in, &v), in, &v)

	terr, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	require.True(t, ok)
	require.Equal(t, "specs.0.software.packages", terr.Field)
	// The prefix is correct as far as it goes, which is what callers render.
	require.NotEmpty(t, terr.Field, "an empty path is the regression this package fixes")
}

func TestEnrichIgnoresUnrelatedErrors(t *testing.T) {
	data := []byte(`{}`)
	v := &root{}

	require.NoError(t, Enrich(nil, data, v))

	sentinel := errors.New("boom")
	require.Equal(t, sentinel, Enrich(sentinel, data, v))

	// Syntax errors carry an offset rather than a path, and are not UnmarshalTypeErrors.
	syntax := jsonv1.Unmarshal([]byte(`{`), v)
	require.Equal(t, syntax, Enrich(syntax, []byte(`{`), v))
}

func TestEnrichIsANoOpWhenPositionCannotBeRecovered(t *testing.T) {
	// If the second decode disagrees (here: nothing wrong with the input), the original error survives
	// untouched rather than being annotated with a misleading path.
	original := &jsonv1.UnmarshalTypeError{Value: "string", Type: nil}
	err := Enrich(original, []byte(`{"inner":{"plain":"fine"}}`), &root{})
	require.Same(t, original, err)
	require.Empty(t, original.Field)
}

func TestFieldPath(t *testing.T) {
	terr, _ := decode(t, `{"specs":[{"custom":"nope"}]}`)
	require.Equal(t, "specs.0.custom", FieldPath(terr))

	require.Equal(t, "a.b", FieldPath(&jsonv1.UnmarshalTypeError{Field: "a.b"}))
	require.Empty(t, FieldPath(errors.New("boom")))
	require.Empty(t, FieldPath(nil))
}
