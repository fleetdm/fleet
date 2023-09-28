package optjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   String
			marshalAs string
		}{
			{`"foo"`, "", String{Set: true, Valid: true, Value: "foo"}, `"foo"`},
			{`""`, "", String{Set: true, Valid: true, Value: ""}, `""`},
			{`null`, "", String{Set: true, Valid: false, Value: ""}, `null`},
			{`123`, "cannot unmarshal number into Go value of type string", String{Set: true, Valid: false, Value: ""}, `null`},
			{`{"v": "foo"}`, "cannot unmarshal object into Go value of type string", String{Set: true, Valid: false, Value: ""}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var s String
				err := json.Unmarshal([]byte(c.data), &s)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, s)

				b, err := json.Marshal(s)
				require.NoError(t, err)
				require.Equal(t, c.marshalAs, string(b))
			})
		}
	})

	t.Run("struct", func(t *testing.T) {
		type N struct {
			S2 String `json:"s2"`
		}
		type T struct {
			I int    `json:"i"`
			S String `json:"s"`
			N N      `json:"n"`
		}

		cases := []struct {
			data      string
			wantErr   string
			wantRes   T
			marshalAs string
		}{
			{`{}`, "", T{}, `{"i": 0, "s": null, "n": {"s2": null}}`},
			{`{"x": "nope"}`, "", T{}, `{"i": 0, "s": null, "n": {"s2": null}}`},
			{`{"i": 1, "s": "a"}`, "", T{I: 1, S: String{Set: true, Valid: true, Value: "a"}}, `{"i": 1, "s": "a", "n": {"s2": null}}`},
			{`{"i": 1, "s": null, "n": {}}`, "", T{I: 1, S: String{Set: true, Valid: false, Value: ""}}, `{"i": 1, "s": null, "n": {"s2": null}}`},
			{`{"i": 1, "s": "a", "n": {"s2": "b"}}`, "", T{I: 1, S: String{Set: true, Valid: true, Value: "a"}, N: N{S2: String{Set: true, Valid: true, Value: "b"}}}, `{"i": 1, "s": "a", "n": {"s2": "b"}}`},
			{`{"i": 1, "s": "a", "n": {"s2": null}}`, "", T{I: 1, S: String{Set: true, Valid: true, Value: "a"}, N: N{S2: String{Set: true, Valid: false, Value: ""}}}, `{"i": 1, "s": "a", "n": {"s2": null}}`},
			{`{"i": 1, "s": true}`, "cannot unmarshal bool into Go struct", T{I: 1, S: String{Set: true, Valid: false, Value: ""}}, `{"i": 1, "s": null, "n": {"s2": null}}`},
			{`{"i": 1, "n": {"s2": 123}}`, "cannot unmarshal number into Go struct", T{I: 1, N: N{S2: String{Set: true, Valid: false, Value: ""}}}, `{"i": 1, "s": null, "n": {"s2": null}}`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var tt T
				err := json.Unmarshal([]byte(c.data), &tt)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, tt)

				b, err := json.Marshal(tt)
				require.NoError(t, err)
				require.JSONEq(t, c.marshalAs, string(b))
			})
		}
	})
}

func TestSlice(t *testing.T) {
	t.Run("slice of ints", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Slice[int]
			marshalAs string
		}{
			{`[1,2,3]`, "", SetSlice([]int{1, 2, 3}), `[1,2,3]`},
			{`[]`, "", SetSlice([]int{}), `[]`},
			{`null`, "", Slice[int]{Set: true, Valid: false, Value: []int{}}, `null`},
			{`[1,"2",3]`, "cannot unmarshal string", Slice[int]{Set: true, Valid: false, Value: nil}, `null`},
			{`123`, "cannot unmarshal number", Slice[int]{Set: true, Valid: false, Value: []int(nil)}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var s Slice[int]
				err := json.Unmarshal([]byte(c.data), &s)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, s)

				b, err := json.Marshal(s)
				require.NoError(t, err)
				require.Equal(t, c.marshalAs, string(b))
			})
		}
	})

	t.Run("slice of strings", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Slice[string]
			marshalAs string
		}{
			{`["foo", "bar"]`, "", SetSlice([]string{"foo", "bar"}), `["foo","bar"]`},
			{`[""]`, "", SetSlice([]string{""}), `[""]`},
			{`null`, "", Slice[string]{Set: true, Valid: false, Value: []string{}}, `null`},
			{`["foo", 123]`, "cannot unmarshal number", Slice[string]{Set: true, Valid: false, Value: []string(nil)}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var s Slice[string]
				err := json.Unmarshal([]byte(c.data), &s)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, s)

				b, err := json.Marshal(s)
				require.NoError(t, err)
				require.Equal(t, c.marshalAs, string(b))
			})
		}
	})

	t.Run("slice of bools", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Slice[bool]
			marshalAs string
		}{
			{`[true, false]`, "", SetSlice([]bool{true, false}), `[true,false]`},
			{`[true, "false"]`, "cannot unmarshal string", Slice[bool]{Set: true, Valid: false, Value: []bool(nil)}, `null`},
			{`null`, "", Slice[bool]{Set: true, Valid: false, Value: []bool{}}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var s Slice[bool]
				err := json.Unmarshal([]byte(c.data), &s)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, s)

				b, err := json.Marshal(s)
				require.NoError(t, err)
				require.Equal(t, c.marshalAs, string(b))
			})
		}
	})
}

func TestSliceWithinStruct(t *testing.T) {
	type Nested struct {
		Numbers Slice[int]    `json:"numbers"`
		Words   Slice[string] `json:"words"`
		Flags   Slice[bool]   `json:"flags"`
	}

	type Parent struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Nested Nested `json:"nested"`
	}

	t.Run("struct", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Parent
			marshalAs string
		}{
			{`{}`, "", Parent{}, `{"id": 0, "name": "", "nested": {"numbers": null, "words": null, "flags": null}}`},
			{`{"id": 1, "name": "test", "nested": {"numbers": [1, 2, 3], "words": ["one", "two"], "flags": [true, false]}}`, "", Parent{
				ID:   1,
				Name: "test",
				Nested: Nested{
					Numbers: SetSlice([]int{1, 2, 3}),
					Words:   SetSlice([]string{"one", "two"}),
					Flags:   SetSlice([]bool{true, false}),
				},
			}, `{"id": 1, "name": "test", "nested": {"numbers": [1,2,3], "words": ["one","two"], "flags": [true,false]}}`},
			{`{"id": 1, "name": "test", "nested": {"numbers": null, "words": ["one", "two"], "flags": [true, false]}}`, "", Parent{
				ID:   1,
				Name: "test",
				Nested: Nested{
					Numbers: Slice[int]{Set: true, Valid: false, Value: []int{}},
					Words:   SetSlice([]string{"one", "two"}),
					Flags:   SetSlice([]bool{true, false}),
				},
			}, `{"id": 1, "name": "test", "nested": {"numbers": null, "words": ["one","two"], "flags": [true,false]}}`},
			{`{"id": 1, "name": "test", "nested": {"numbers": [1, 2, 3], "words": null, "flags": [true, false]}}`, "", Parent{
				ID:   1,
				Name: "test",
				Nested: Nested{
					Numbers: SetSlice([]int{1, 2, 3}),
					Words:   Slice[string]{Set: true, Valid: false, Value: []string{}},
					Flags:   SetSlice([]bool{true, false}),
				},
			}, `{"id": 1, "name": "test", "nested": {"numbers": [1,2,3], "words": null, "flags": [true,false]}}`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var p Parent
				err := json.Unmarshal([]byte(c.data), &p)

				if c.wantErr != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, p)

				b, err := json.Marshal(p)
				require.NoError(t, err)
				require.JSONEq(t, c.marshalAs, string(b))
			})
		}
	})
}
