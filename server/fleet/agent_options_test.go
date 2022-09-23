package fleet

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAgentOptions(t *testing.T) {
	cases := []struct {
		desc    string
		in      string
		wantErr string
	}{
		{"empty object", "{}", ""},
		{"empty config", `{"config":{}}`, ""},
		{"empty overrides", `{"overrides":{}}`, ""},

		{"unknown top-level key", `{"foo":1}`, `unknown field "foo"`},
		{"unknown config key", `{"config":{"foo":1}}`, `unknown field "foo"`},
		{"unknown overrides key", `{"overrides": {"foo": 1}}`, `unknown field "foo"`},
		{"unknown overrides config key", `{"overrides": {
			"platforms": {
				"linux": {"foo":1}
			}
		}}`, `unknown field "foo"`},

		{"extra top-level bytes", `{}true`, `extra bytes`},
		{"extra config bytes", `{"config":{}true}`, `invalid character 't' after object`},
		{"extra overrides bytes", `{"overrides":{}true}`, `invalid character 't' after object`},

		{"valid config", `{"config":{
			"options": {"aws_debug": true, "events_max": 3},
			"views": {"view1": "select 1"}
		}}`, ""},
		{"valid overrides", `{"overrides":{
			"platforms": {
				"linux": {
					"options": {"aws_debug": true, "events_max": 3},
					"views": {"view1": "select 1"}
				},
				"darwin": {
					"options": {"aws_debug": false, "events_max": 1},
					"views": {"view2": "select 2"}
				}
			}
		}}`, ""},

		{"invalid config value", `{"config":{
			"events": {
				"disable_subscribers": true
			},
			"options": {"aws_debug": 1}
		}}`, "cannot unmarshal bool into Go struct field .events.disable_subscribers of type []string"},
		{"invalid overrides value", `{"overrides":{
			"platforms": {
				"linux": {
					"options": {"aws_debug": true, "events_max": "nope"}
				}
			}
		}}`, `cannot unmarshal string into Go struct field osqueryOptions.options.events_max of type uint64`},

		{"valid packs string", `{"config":{
			"packs": {
				"pack1": "ok"
			}
		}}`, ""},
		{"valid packs object", `{"config":{
			"packs": {
				"pack1": {
					"schedule": {
						"1000": {
							"query": "select 1"
						}
					},
					"platform": "darwin"
				}
			}
		}}`, ""},
		{"invalid packs object key", `{"config":{
			"packs": {
				"pack1": {
					"schedule": {
						"1000": {
							"query": "select 1",
							"foo": 2
						}
					},
					"platform": "darwin"
				}
			}
		}}`, `unknown field "foo"`},
		{"invalid packs type", `{"config":{
			"packs": {
				"pack1": 1
			}
		}}`, `invalid number value`},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := ValidateJSONAgentOptions([]byte(c.in))
			t.Logf("%T", errors.Unwrap(err))
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
