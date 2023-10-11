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

func TestBool(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
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
