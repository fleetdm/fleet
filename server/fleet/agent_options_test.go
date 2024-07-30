package fleet

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAgentOptions(t *testing.T) {
	cases := []struct {
		desc      string
		in        string
		isPremium bool
		wantErr   string
	}{
		{"empty object", "{}", true, ""},
		{"empty config", `{"config":{}}`, true, ""},
		{"empty overrides", `{"overrides":{}}`, true, ""},

		{"unknown top-level key", `{"foo":1}`, true, `unknown field "foo"`},
		{"unknown config key", `{"config":{"foo":1}}`, true, `unknown field "foo"`},
		{"unknown overrides key", `{"overrides": {"foo": 1}}`, true, `unknown field "foo"`},
		{"unknown overrides config key", `{"overrides": {
			"platforms": {
				"linux": {"foo":1}
			}
		}}`, true, `unknown field "foo"`},

		{"valid script timeout", `{"script_execution_timeout": 600}`, true, ""},

		{"invalid script timeout", `{"script_execution_timeout": 3601}`, true, `script_execution_timeout' value exceeds limit. Maximum value is 3600`},

		{"overrides.platform is null", `{"overrides": {
			"platforms": {
				"darwin": null
			}
		}}`, true, `platforms cannot be null. To remove platform overrides omit overrides from agent options.`},

		{"extra top-level bytes", `{}true`, true, `extra bytes`},
		{"extra config bytes", `{"config":{}true}`, true, `invalid character 't' after object`},
		{"extra overrides bytes", `{"overrides":{}true}`, true, `invalid character 't' after object`},

		{"valid config", `{"config":{
			"options": {"aws_debug": true, "events_max": 3},
			"views": {"view1": "select 1"}
		}}`, true, ""},
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
		}}`, true, ""},

		{"invalid config value", `{"config":{
			"events": {
				"disable_subscribers": true
			},
			"options": {"aws_debug": 1}
		}}`, true, "cannot unmarshal bool into Go struct field .events.disable_subscribers of type []string"},
		{"invalid overrides value", `{"overrides":{
			"platforms": {
				"linux": {
					"options": {"aws_debug": true, "events_max": "nope"}
				}
			}
		}}`, true, `cannot unmarshal string into Go struct field osqueryOptions.options.events_max of type uint64`},

		{"valid packs string", `{"config":{
			"packs": {
				"pack1": "ok"
			}
		}}`, true, ""},
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
		}}`, true, ""},
		{"invalid packs object key is accepted as we do not validate packs", `{"config":{
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
		}}`, true, ``},
		{"invalid packs type is accepted as we do not validate packs", `{"config":{
			"packs": {
				"pack1": 1
			}
		}}`, true, ``},
		{"invalid schedule type is accepted as we do not validate schedule", `{"config":{
			"schedule": {
				"foo": 1
			}
		}}`, true, ``},
		{"option added in osquery 5.5.1", `{"config":{
			"options": {
				"malloc_trim_threshold": 100
			}
		}}`, true, ``},
		{"option removed in osquery 5.5.1", `{"config":{
			"options": {
				"yara_malloc_trim": true
			}
		}}`, true, `unknown field "yara_malloc_trim"`},
		{
			"option added in osquery 5.11.0", `{"config":{
			"options": {
				"keychain_access_cache": true
			}
		}}`, true, ``,
		},
		{"valid command-line flag", `{"command_line_flags":{
			"alarm_timeout": 1
		}}`, true, ``},
		{"invalid command-line flag", `{"command_line_flags":{
			"no_such_flag": true
		}}`, true, `unknown field "no_such_flag"`},
		{"invalid command-line value", `{"command_line_flags":{
			"enable_tables": 123
		}}`, true, `cannot unmarshal number into Go struct field osqueryCommandLineFlags.enable_tables of type string`},
		{"setting a valid os-specific flag", `{"command_line_flags":{
			"users_service_delay": 123
		}}`, true, ``},
		{"setting a valid os-specific option", `{"config":{
			"options": {
				"users_service_delay": 123
			}
		}}`, true, ``},
		{"setting an invalid value for an os-specific flag", `{"command_line_flags":{
			"disable_endpointsecurity": "ok"
		}}`, true, `command-line flags: json: cannot unmarshal string into Go struct field osqueryCommandLineFlags.disable_endpointsecurity of type bool`},
		{"setting an invalid value for an os-specific option", `{"config":{
			"options": {
				"disable_endpointsecurity": "ok"
			}
		}}`, true, `common config: json: cannot unmarshal string into Go struct field osqueryOptions.options.disable_endpointsecurity of type bool`},
		{"setting an empty update_channels", `{
			"update_channels": null
		}`, true, `update_channels cannot be null`},
		{"setting a empty channel in update_channels", `{
			"update_channels": {
				"osqueryd": "5.10.2",
				"orbit": null
			}
		}`, true, `update_channels.orbit is defined but not set`},
		{"setting a channel in update_channels to empty string", `{
			"update_channels": {
				"osqueryd": "5.10.2",
				"orbit": ""
			}
		}`, true, `update_channels.orbit is set to an empty string`},
		{"setting a channel to unknown component in update_channels", `{
			"update_channels": {
				"osqueryd": "5.10.2",
				"unknown": "foobar"
			}
		}`, true, `update_channels: json: unknown field "unknown"`},
		{"setting update_channels non-premium", `{
			"update_channels": {
				"osqueryd": "5.10.2",
				"orbit": "foobar"
			}
		}`, false, `Requires Fleet Premium license`},
		{"setting update_channels", `{
			"update_channels": {
				"osqueryd": "5.10.2",
				"orbit": "foobar"
			}
		}`, true, ``},
		{"setting osquery 5.12.X flag in config.options and command_line_flags", `{
			"config": {
				"options": {
					"logger_tls_backoff_max": 100
				}
			},
			"command_line_flags": {
				"logger_tls_backoff_max": 200
			} 
		}`, true, ``},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := ValidateJSONAgentOptions(context.Background(), nil, []byte(c.in), c.isPremium)
			t.Logf("%T", errors.Unwrap(err))
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
