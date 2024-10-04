package optjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestBool(t *testing.T) {
	t.Run("plain bool", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Bool
			marshalAs string
		}{
			{`true`, "", Bool{Set: true, Valid: true, Value: true}, `true`},
			{`null`, "", Bool{Set: true, Valid: false, Value: false}, `null`},
			{`123`, "cannot unmarshal number into Go value of type bool", Bool{Set: true, Valid: false, Value: false}, `null`},
			{`{"v": "foo"}`, "cannot unmarshal object into Go value of type bool", Bool{Set: true, Valid: false, Value: false}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var s Bool
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
			B2 Bool `json:"b2"`
		}
		type T struct {
			I int  `json:"i"`
			B Bool `json:"b"`
			N N    `json:"n"`
		}

		cases := []struct {
			data      string
			wantErr   string
			wantRes   T
			marshalAs string
		}{
			{`{}`, "", T{}, `{"i": 0, "b": null, "n": {"b2": null}}`},
			{`{"x": "nope"}`, "", T{}, `{"i": 0, "b": null, "n": {"b2": null}}`},
			{`{"i": 1, "b": true}`, "", T{I: 1, B: Bool{Set: true, Valid: true, Value: true}}, `{"i": 1, "b": true, "n": {"b2": null}}`},
			{`{"i": 1, "b": null, "n": {}}`, "", T{I: 1, B: Bool{Set: true, Valid: false, Value: false}}, `{"i": 1, "b": null, "n": {"b2": null}}`},
			{`{"i": 1, "b": false, "n": {"b2": true}}`, "", T{I: 1, B: Bool{Set: true, Valid: true, Value: false}, N: N{B2: Bool{Set: true, Valid: true, Value: true}}}, `{"i": 1, "b": false, "n": {"b2": true}}`},
			{`{"i": 1, "b": true, "n": {"b2": null}}`, "", T{I: 1, B: Bool{Set: true, Valid: true, Value: true}, N: N{B2: Bool{Set: true, Valid: false, Value: false}}}, `{"i": 1, "b": true, "n": {"b2": null}}`},
			{`{"i": 1, "b": ""}`, "cannot unmarshal string into Go struct", T{I: 1, B: Bool{Set: true, Valid: false, Value: false}}, `{"i": 1, "b": null, "n": {"b2": null}}`},
			{`{"i": 1, "n": {"b2": 123}}`, "cannot unmarshal number into Go struct", T{I: 1, N: N{B2: Bool{Set: true, Valid: false, Value: false}}}, `{"i": 1, "b": null, "n": {"b2": null}}`},
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

func TestInt(t *testing.T) {
	t.Run("plain int", func(t *testing.T) {
		cases := []struct {
			data      string
			wantErr   string
			wantRes   Int
			marshalAs string
		}{
			{`1`, "", Int{Set: true, Valid: true, Value: 1}, `1`},
			{`-1`, "", Int{Set: true, Valid: true, Value: -1}, `-1`},
			{`0`, "", Int{Set: true, Valid: true, Value: 0}, `0`},
			{`1.23`, "cannot unmarshal number 1.23 into Go value of type int", Int{Set: true, Valid: false, Value: 0}, `null`},
			{`null`, "", Int{Set: true, Valid: false, Value: 0}, `null`},
			{`"x"`, "cannot unmarshal string into Go value of type int", Int{Set: true, Valid: false, Value: 0}, `null`},
			{`{"v": "foo"}`, "cannot unmarshal object into Go value of type int", Int{Set: true, Valid: false, Value: 0}, `null`},
		}

		for _, c := range cases {
			t.Run(c.data, func(t *testing.T) {
				var i Int
				err := json.Unmarshal([]byte(c.data), &i)

				if c.wantErr != "" {
					require.Error(t, err)
					require.ErrorContains(t, err, c.wantErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, c.wantRes, i)

				b, err := json.Marshal(i)
				require.NoError(t, err)
				require.Equal(t, c.marshalAs, string(b))
			})
		}
	})

	t.Run("struct", func(t *testing.T) {
		type N struct {
			I2 Int `json:"i2"`
		}
		type T struct {
			I Int  `json:"i"`
			B bool `json:"b"`
			N N    `json:"n"`
		}

		cases := []struct {
			data      string
			wantErr   string
			wantRes   T
			marshalAs string
		}{
			{`{}`, "", T{}, `{"i": null, "b": false, "n": {"i2": null}}`},
			{`{"x": "nope"}`, "", T{}, `{"i": null, "b": false, "n": {"i2": null}}`},
			{`{"i": 1, "b": true}`, "", T{I: Int{Set: true, Valid: true, Value: 1}, B: true}, `{"i": 1, "b": true, "n": {"i2": null}}`},
			{`{"i": null, "b": true, "n": {}}`, "", T{I: Int{Set: true, Valid: false, Value: 0}, B: true}, `{"i": null, "b": true, "n": {"i2": null}}`},
			{`{"i": 1, "b": true, "n": {"i2": 2}}`, "", T{I: Int{Set: true, Valid: true, Value: 1}, B: true, N: N{I2: Int{Set: true, Valid: true, Value: 2}}}, `{"i": 1, "b": true, "n": {"i2": 2}}`},
			{`{"i": 1, "b": true, "n": {"i2": null}}`, "", T{I: Int{Set: true, Valid: true, Value: 1}, B: true, N: N{I2: Int{Set: true, Valid: false, Value: 0}}}, `{"i": 1, "b": true, "n": {"i2": null}}`},
			{`{"i": "", "b": true}`, "cannot unmarshal string into Go struct", T{I: Int{Set: true, Valid: false, Value: 0}, B: false}, `{"i": null, "b": false, "n": {"i2": null}}`},
			{`{"b": true, "n": {"i2": true}}`, "cannot unmarshal bool into Go struct", T{I: Int{Set: false, Valid: false, Value: 0}, B: true, N: N{I2: Int{Set: true, Valid: false, Value: 0}}}, `{"i": null, "b": true, "n": {"i2": null}}`},
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
			{data: `[1,2,3]`, wantErr: "", wantRes: SetSlice([]int{1, 2, 3}), marshalAs: `[1,2,3]`},
			{data: `[]`, wantErr: "", wantRes: SetSlice([]int{}), marshalAs: `[]`},
			{data: `null`, wantErr: "", wantRes: Slice[int]{Set: true, Valid: false, Value: []int{}}, marshalAs: `null`},
			{data: `[1,"2",3]`, wantErr: "cannot unmarshal string", wantRes: Slice[int]{Set: true, Valid: false, Value: nil}, marshalAs: `null`},
			{data: `123`, wantErr: "cannot unmarshal number", wantRes: Slice[int]{Set: true, Valid: false, Value: []int(nil)}, marshalAs: `null`},
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
			{data: `["foo", "bar"]`, wantErr: "", wantRes: SetSlice([]string{"foo", "bar"}), marshalAs: `["foo","bar"]`},
			{data: `[""]`, wantErr: "", wantRes: SetSlice([]string{""}), marshalAs: `[""]`},
			{data: `null`, wantErr: "", wantRes: Slice[string]{Set: true, Valid: false, Value: []string{}}, marshalAs: `null`},
			{data: `["foo", 123]`, wantErr: "cannot unmarshal number", wantRes: Slice[string]{Set: true, Valid: false, Value: []string(nil)}, marshalAs: `null`},
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
			{data: `[true, false]`, wantErr: "", wantRes: SetSlice([]bool{true, false}), marshalAs: `[true,false]`},
			{data: `[true, "false"]`, wantErr: "cannot unmarshal string", wantRes: Slice[bool]{Set: true, Valid: false, Value: []bool(nil)}, marshalAs: `null`},
			{data: `null`, wantErr: "", wantRes: Slice[bool]{Set: true, Valid: false, Value: []bool{}}, marshalAs: `null`},
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
			{data: `{}`, wantErr: "", wantRes: Parent{}, marshalAs: `{"id": 0, "name": "", "nested": {"numbers": null, "words": null, "flags": null}}`},
			{
				data:    `{"id": 1, "name": "test", "nested": {"numbers": [1, 2, 3], "words": ["one", "two"], "flags": [true, false]}}`,
				wantErr: "",
				wantRes: Parent{
					ID:   1,
					Name: "test",
					Nested: Nested{
						Numbers: SetSlice([]int{1, 2, 3}),
						Words:   SetSlice([]string{"one", "two"}),
						Flags:   SetSlice([]bool{true, false}),
					},
				},
				marshalAs: `{"id": 1, "name": "test", "nested": {"numbers": [1,2,3], "words": ["one","two"], "flags": [true,false]}}`,
			},
			{
				data:    `{"id": 1, "name": "test", "nested": {"numbers": null, "words": ["one", "two"], "flags": [true, false]}}`,
				wantErr: "",
				wantRes: Parent{
					ID:   1,
					Name: "test",
					Nested: Nested{
						Numbers: Slice[int]{Set: true, Valid: false, Value: []int{}},
						Words:   SetSlice([]string{"one", "two"}),
						Flags:   SetSlice([]bool{true, false}),
					},
				},
				marshalAs: `{"id": 1, "name": "test", "nested": {"numbers": null, "words": ["one","two"], "flags": [true,false]}}`,
			},
			{
				data:    `{"id": 1, "name": "test", "nested": {"numbers": [1, 2, 3], "words": null, "flags": [true, false]}}`,
				wantErr: "",
				wantRes: Parent{
					ID:   1,
					Name: "test",
					Nested: Nested{
						Numbers: SetSlice([]int{1, 2, 3}),
						Words:   Slice[string]{Set: true, Valid: false, Value: []string{}},
						Flags:   SetSlice([]bool{true, false}),
					},
				},
				marshalAs: `{"id": 1, "name": "test", "nested": {"numbers": [1,2,3], "words": null, "flags": [true,false]}}`,
			},
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

func TestAny(t *testing.T) {
	t.Parallel()
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	SetItem := func(item Item) Any[Item] {
		return Any[Item]{Set: true, Valid: true, Value: item}
	}

	cases := []struct {
		data      string
		wantErr   string
		wantRes   Any[Item]
		marshalAs string
	}{
		{data: `{ "id": 1, "name": "bozo" }`, wantErr: "", wantRes: SetItem(Item{ID: 1, Name: "bozo"}),
			marshalAs: `{"id":1,"name":"bozo"}`},
		{data: `null`, wantErr: "", wantRes: Any[Item]{Set: true, Valid: false}, marshalAs: `null`},
		{data: `[]`, wantErr: "cannot unmarshal array", wantRes: Any[Item]{Set: true, Valid: false, Value: Item{}}, marshalAs: `null`},
	}

	for _, c := range cases {
		t.Run(c.data, func(t *testing.T) {
			var s Any[Item]
			err := json.Unmarshal([]byte(c.data), &s)

			if c.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.wantRes, s)

			b, err := json.Marshal(s)
			require.NoError(t, err)
			require.Equal(t, c.marshalAs, string(b))
		})
	}
}
