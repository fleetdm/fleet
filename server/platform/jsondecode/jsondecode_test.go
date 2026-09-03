package jsondecode

import (
	jsonv1 "encoding/json"
	"errors"
	"io"
	"strings"
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

func decode(t *testing.T, in string) (*jsonv1.UnmarshalTypeError, error) {
	t.Helper()
	var v root
	err := Unmarshal([]byte(in), &v)
	terr, _ := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	return terr, err
}

func TestReportsPathLostByCustomUnmarshaler(t *testing.T) {
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

func TestMatchesEncodingJSONForPlainFields(t *testing.T) {
	// Plain fields never lost their path under Go 1.27, so the translation must reproduce exactly what
	// encoding/json reports for them: same Value and Type, same field path.
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

// TestStopsAtCompositeUnmarshalerBoundary documents a known limit. Locating a failure relies on v2
// error semantics, and v2 cannot see inside a custom UnmarshalJSON: it reports the position of the
// value handed to the method, not of the field within it. For a scalar wrapper such as optjson.Int that
// is the whole path, but for a composite such as optjson.Slice the path stops at the slice.
//
// Go 1.26 reported the full "specs.software.packages.self_service" here, because the v1 decoder
// accumulated its field stack across the custom unmarshaler. Recovering that last segment again would
// mean giving the optjson composites a v2 UnmarshalJSONFrom method, which changes how they decode
// everywhere v2 is used, so it is deliberately out of scope here. A truncated-but-correct prefix is
// still a large improvement on the empty path this package exists to fix.
func TestStopsAtCompositeUnmarshalerBoundary(t *testing.T) {
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

	var v request
	err := Unmarshal([]byte(`{"specs":[{"software":{"packages":[{"self_service":"yes"}]}}]}`), &v)

	terr, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	require.True(t, ok)
	require.Equal(t, "specs.0.software.packages", terr.Field)
	// The prefix is correct as far as it goes, which is what callers render.
	require.NotEmpty(t, terr.Field, "an empty path is the regression this package fixes")
}

func TestPassesThroughNonSemanticErrors(t *testing.T) {
	// Syntax errors and io.EOF have to reach callers unchanged: JSONStrictDecode relies on io.EOF to
	// detect trailing content, and nothing treats malformed JSON as a type error.
	syntax := Unmarshal([]byte(`{`), &root{})
	require.Error(t, syntax)
	require.False(t, IsTypeError(syntax))
	require.Empty(t, FieldPath(syntax))

	require.NoError(t, Unmarshal([]byte(`{}`), &root{}))
}

func TestUnknownFieldKeepsV1Wording(t *testing.T) {
	// server/service/teams.go, server/fleet/agent_options.go and pkg/spec all detect this case by
	// matching encoding/json's exact message, so it has to survive verbatim.
	err := Unmarshal([]byte(`{"nope":1}`), &root{}, RejectUnknownMembers())
	require.EqualError(t, err, `json: unknown field "nope"`)

	// ...and only when asked for: unknown members are ignored by default, as json.Unmarshal does.
	require.NoError(t, Unmarshal([]byte(`{"nope":1}`), &root{}))
}

func TestPreservesEncodingJSONDecodingBehavior(t *testing.T) {
	t.Run("member matching stays case-insensitive", func(t *testing.T) {
		var v root
		require.NoError(t, Unmarshal([]byte(`{"INNER":{"PLAIN":"x"}}`), &v))
		require.Equal(t, "x", v.Inner.Plain)
	})

	t.Run("duplicate names allowed", func(t *testing.T) {
		var v root
		require.NoError(t, Unmarshal([]byte(`{"inner":{"plain":"a"},"inner":{"plain":"b"}}`), &v))
	})

	t.Run("Decode leaves trailing bytes, Unmarshal rejects them", func(t *testing.T) {
		r := strings.NewReader(`{"inner":{"plain":"a"}}{"inner":{"plain":"b"}}`)
		dec := NewDecoder(r)

		var first root
		require.NoError(t, dec.Decode(&first))
		require.Equal(t, "a", first.Inner.Plain)

		// A second Decode reaches the next value, and a third reports io.EOF.
		var second root
		require.NoError(t, dec.Decode(&second))
		require.Equal(t, "b", second.Inner.Plain)
		require.ErrorIs(t, dec.Decode(&second), io.EOF)

		require.Error(t, Unmarshal([]byte(`{"inner":{}}{"inner":{}}`), &root{}))
	})
}

func TestReportsPathForPlainFields(t *testing.T) {
	for _, c := range []struct {
		name     string
		in       string
		wantPath string
	}{
		{"plain field", `{"inner":{"plain":123}}`, "inner.plain"},
		{"custom unmarshaler", `{"inner":{"custom":"nope"}}`, "inner.custom"},
		{"slice element", `{"specs":[{"custom":"nope"}]}`, "specs.0.custom"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := Unmarshal([]byte(c.in), &root{})
			require.Error(t, err)
			require.True(t, IsTypeError(err))
			require.Equal(t, c.wantPath, FieldPath(err))
		})
	}

	require.NoError(t, Unmarshal([]byte(`{"inner":{"plain":"fine"}}`), &root{}))
}

func TestIsTypeError(t *testing.T) {
	require.True(t, IsTypeError(Unmarshal([]byte(`{"inner":{"plain":123}}`), &root{})))
	// Also recognizes an error from a plain encoding/json decode, for callers that mix the two.
	var v root
	require.True(t, IsTypeError(jsonv1.Unmarshal([]byte(`{"inner":{"plain":123}}`), &v)))

	// Malformed JSON is a syntax problem, not a type problem.
	require.False(t, IsTypeError(Unmarshal([]byte(`{`), &root{})))
	require.False(t, IsTypeError(errors.New("boom")))
	require.False(t, IsTypeError(nil))
}

func TestFieldPath(t *testing.T) {
	terr, _ := decode(t, `{"specs":[{"custom":"nope"}]}`)
	require.Equal(t, "specs.0.custom", FieldPath(terr))

	require.Equal(t, "a.b", FieldPath(&jsonv1.UnmarshalTypeError{Field: "a.b"}))
	require.Empty(t, FieldPath(errors.New("boom")))
	require.Empty(t, FieldPath(nil))
}
