package optjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringOr(t *testing.T) {
	type child struct {
		Name string `json:"name"`
	}

	type target struct {
		Field StringOr[[]string] `json:"field"`
		Array []StringOr[*child] `json:"array"`
	}

	type nested struct {
		Inception StringOr[*target] `json:"inception"`
	}

	cases := []struct {
		name         string
		getVar       func() any
		src          string // json source to unmarshal into the value returned by getVar
		marshalAs    string // how the value should marshal back to json
		unmarshalErr string // if non-empty, unmarshal should fail with this error
	}{
		{
			name:      "simple string",
			getVar:    func() any { var s StringOr[int]; return &s },
			src:       `"abc"`,
			marshalAs: `"abc"`,
		},
		{
			name:      "simple integer",
			getVar:    func() any { var s StringOr[int]; return &s },
			src:       `123`,
			marshalAs: `123`,
		},
		{
			name:         "invalid bool",
			getVar:       func() any { var s StringOr[int]; return &s },
			src:          `true`,
			unmarshalErr: "cannot unmarshal bool into Go value of type int",
		},
		{
			name:      "field string",
			getVar:    func() any { var s target; return &s },
			src:       `{"field":"abc"}`,
			marshalAs: `{"field":"abc", "array": null}`,
		},
		{
			name:      "field strings",
			getVar:    func() any { var s target; return &s },
			src:       `{"field":["a", "b", "c"]}`,
			marshalAs: `{"field":["a", "b", "c"], "array": null}`,
		},
		{
			name:      "field empty array",
			getVar:    func() any { var s target; return &s },
			src:       `{"field":[]}`,
			marshalAs: `{"field":[], "array": null}`,
		},
		{
			name:         "field invalid object",
			getVar:       func() any { var s target; return &s },
			src:          `{"field":{}}`,
			unmarshalErr: "cannot unmarshal object into Go struct field target.field of type []string",
		},
		{
			name:      "array field null",
			getVar:    func() any { var s target; return &s },
			src:       `{"array":null}`,
			marshalAs: `{"array":null, "field": ""}`,
		},
		{
			name:      "array field empty",
			getVar:    func() any { var s target; return &s },
			src:       `{"array":[]}`,
			marshalAs: `{"array":[], "field": ""}`,
		},
		{
			name:      "array field single",
			getVar:    func() any { var s target; return &s },
			src:       `{"array":["a"]}`,
			marshalAs: `{"array":["a"], "field": ""}`,
		},
		{
			name:      "array field empty child",
			getVar:    func() any { var s target; return &s },
			src:       `{"array":["a", {}]}`,
			marshalAs: `{"array":["a", {"name":""}], "field": ""}`,
		},
		{
			name:      "array field set child",
			getVar:    func() any { var s target; return &s },
			src:       `{"array":["a", {"name": "x"}]}`,
			marshalAs: `{"array":["a", {"name":"x"}], "field": ""}`,
		},
		{
			name:      "inception string",
			getVar:    func() any { var s nested; return &s },
			src:       `{"inception":"a"}`,
			marshalAs: `{"inception":"a"}`,
		},
		{
			name:      "inception target field",
			getVar:    func() any { var s nested; return &s },
			src:       `{"inception":{"field":["a", "b"]}}`,
			marshalAs: `{"inception":{"field":["a", "b"], "array": null}}`,
		},
		{
			name:      "inception target field and array",
			getVar:    func() any { var s nested; return &s },
			src:       `{"inception":{"field":["a", "b"], "array": ["c", {"name": "x"}]}}`,
			marshalAs: `{"inception":{"field":["a", "b"], "array": ["c", {"name": "x"}]}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target := c.getVar()
			err := json.Unmarshal([]byte(c.src), target)
			if c.unmarshalErr != "" {
				require.ErrorContains(t, err, c.unmarshalErr)
				return
			}
			require.NoError(t, err)

			data, err := json.Marshal(target)
			require.NoError(t, err)
			require.JSONEq(t, c.marshalAs, string(data))
		})
	}
}
