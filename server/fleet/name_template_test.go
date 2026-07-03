package fleet

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestValidateHostNameTemplate(t *testing.T) {
	cases := []struct {
		name     string
		tmpl     string
		wantNorm string // expected normalized template on success
		wantErr  string
	}{
		{name: "plain string", tmpl: "workstation", wantNorm: "workstation"},
		{name: "hardware serial", tmpl: "$FLEET_VAR_HOST_HARDWARE_SERIAL", wantNorm: "$FLEET_VAR_HOST_HARDWARE_SERIAL"},
		{name: "uuid braced", tmpl: "${FLEET_VAR_HOST_UUID}", wantNorm: "${FLEET_VAR_HOST_UUID}"},
		{name: "platform", tmpl: "$FLEET_VAR_HOST_PLATFORM", wantNorm: "$FLEET_VAR_HOST_PLATFORM"},
		{
			name:     "mixed",
			tmpl:     "mac-$FLEET_VAR_HOST_HARDWARE_SERIAL-${FLEET_VAR_HOST_UUID}",
			wantNorm: "mac-$FLEET_VAR_HOST_HARDWARE_SERIAL-${FLEET_VAR_HOST_UUID}",
		},
		{
			name:     "surrounding whitespace is trimmed in the returned value",
			tmpl:     "  serial-$FLEET_VAR_HOST_HARDWARE_SERIAL  ",
			wantNorm: "serial-$FLEET_VAR_HOST_HARDWARE_SERIAL",
		},

		{name: "empty", tmpl: "", wantErr: "can't be empty"},
		{name: "whitespace only", tmpl: "   ", wantErr: "can't be empty"},
		{
			name:    "unsupported var",
			tmpl:    "$FLEET_VAR_HOST_END_USER_IDP_GROUPS",
			wantErr: "Fleet variable $FLEET_VAR_HOST_END_USER_IDP_GROUPS is not supported in host name templates.",
		},
		{
			name:    "supported var with unsupported suffix",
			tmpl:    "$FLEET_VAR_HOST_UUID_EXTRA",
			wantErr: "Fleet variable $FLEET_VAR_HOST_UUID_EXTRA is not supported in host name templates.",
		},
		{
			name:    "secret var",
			tmpl:    "$FLEET_SECRET_FOO",
			wantErr: "Secret variables aren't supported in host name templates.",
		},
		{
			name:    "secret var braced",
			tmpl:    "${FLEET_SECRET_FOO}",
			wantErr: "Secret variables aren't supported in host name templates.",
		},
		{name: "tab control char", tmpl: "bad\tname", wantErr: "control characters"},
		{name: "rtl override format char", tmpl: "bad‮name", wantErr: "control characters"},
		{name: "zero-width joiner format char", tmpl: "bad‍name", wantErr: "control characters"},
		{name: "invalid utf-8", tmpl: "bad\xff\xfename", wantErr: "valid UTF-8"},
		{name: "too long", tmpl: strings.Repeat("a", 256), wantErr: "255 characters"},
		{name: "max length ok", tmpl: strings.Repeat("a", 255), wantNorm: strings.Repeat("a", 255)},
		// The limit is 255 characters, not bytes: 255 multi-byte runes must pass.
		{name: "255 multi-byte runes ok", tmpl: strings.Repeat("é", 255), wantNorm: strings.Repeat("é", 255)},
		{name: "256 multi-byte runes too long", tmpl: strings.Repeat("é", 256), wantErr: "255 characters"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			norm, err := ValidateHostNameTemplate(c.tmpl)
			if c.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, c.wantNorm, norm)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), c.wantErr)
			require.Empty(t, norm)
		})
	}
}

func TestResolveHostNameTemplate(t *testing.T) {
	host := &Host{
		HardwareSerial: "C02ABC123",
		UUID:           "1234-5678",
		Platform:       "darwin",
	}

	cases := []struct {
		name string
		tmpl string
		host *Host
		want string
	}{
		{name: "no vars", tmpl: "workstation", host: host, want: "workstation"},
		{name: "serial", tmpl: "$FLEET_VAR_HOST_HARDWARE_SERIAL", host: host, want: "C02ABC123"},
		{name: "uuid braced", tmpl: "${FLEET_VAR_HOST_UUID}", host: host, want: "1234-5678"},
		{name: "platform darwin maps to macos", tmpl: "$FLEET_VAR_HOST_PLATFORM", host: host, want: "macos"},
		{
			name: "mixed and repeated",
			tmpl: "$FLEET_VAR_HOST_PLATFORM-$FLEET_VAR_HOST_HARDWARE_SERIAL-${FLEET_VAR_HOST_HARDWARE_SERIAL}",
			host: host,
			want: "macos-C02ABC123-C02ABC123",
		},
		{
			name: "non-darwin platform unchanged",
			tmpl: "$FLEET_VAR_HOST_PLATFORM",
			host: &Host{Platform: "windows"},
			want: "windows",
		},
		{
			name: "empty host field resolves to empty string",
			tmpl: "serial=$FLEET_VAR_HOST_HARDWARE_SERIAL",
			host: &Host{},
			want: "serial=",
		},
		{
			// A host value that itself contains variable syntax must not be
			// re-substituted by a later pass (single-pass replacement).
			name: "host value containing variable syntax is not re-substituted",
			tmpl: "$FLEET_VAR_HOST_HARDWARE_SERIAL",
			host: &Host{HardwareSerial: "$FLEET_VAR_HOST_UUID", UUID: "real-uuid"},
			want: "$FLEET_VAR_HOST_UUID",
		},
		{
			name: "unsupported longer variable name is left untouched",
			tmpl: "$FLEET_VAR_HOST_UUID_EXTRA",
			host: host,
			want: "$FLEET_VAR_HOST_UUID_EXTRA",
		},
		{name: "nil host leaves template unchanged", tmpl: "$FLEET_VAR_HOST_UUID", host: nil, want: "$FLEET_VAR_HOST_UUID"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, ResolveHostNameTemplate(c.tmpl, c.host))
		})
	}
}

func TestTeamNameTemplateRoundTrip(t *testing.T) {
	t.Run("TeamMDM", func(t *testing.T) {
		in := TeamMDM{NameTemplate: "$FLEET_VAR_HOST_HARDWARE_SERIAL"}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_HARDWARE_SERIAL"`)

		var out TeamMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.Equal(t, in.NameTemplate, out.NameTemplate)
	})

	t.Run("TeamSpecMDM", func(t *testing.T) {
		in := TeamSpecMDM{NameTemplate: optjson.SetString("$FLEET_VAR_HOST_UUID")}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_UUID"`)

		var out TeamSpecMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.True(t, out.NameTemplate.Set)
		require.True(t, out.NameTemplate.Valid)
		require.Equal(t, "$FLEET_VAR_HOST_UUID", out.NameTemplate.Value)
	})

	t.Run("TeamSpecMDM absent", func(t *testing.T) {
		var out TeamSpecMDM
		require.NoError(t, json.Unmarshal([]byte(`{}`), &out))
		require.False(t, out.NameTemplate.Set)
	})

	t.Run("TeamPayloadMDM", func(t *testing.T) {
		in := TeamPayloadMDM{NameTemplate: optjson.SetString("$FLEET_VAR_HOST_PLATFORM")}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_PLATFORM"`)

		var out TeamPayloadMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.True(t, out.NameTemplate.Set)
		require.Equal(t, "$FLEET_VAR_HOST_PLATFORM", out.NameTemplate.Value)
	})

	t.Run("TeamConfig storage round-trip", func(t *testing.T) {
		// name_template rides in the teams.config JSON blob (no dedicated
		// column), so verify it survives the actual SQL Value()/Scan() boundary.
		in := TeamConfig{MDM: TeamMDM{NameTemplate: "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}
		val, err := in.Value()
		require.NoError(t, err)

		var out TeamConfig
		require.NoError(t, out.Scan(val))
		require.Equal(t, "$FLEET_VAR_HOST_HARDWARE_SERIAL", out.MDM.NameTemplate)
	})
}

func TestActivityTypeEditedHostNameTemplate(t *testing.T) {
	require.Equal(t, "edited_host_name_template", ActivityTypeEditedHostNameTemplate{}.ActivityName())

	t.Run("marshal with template", func(t *testing.T) {
		tmpl := "$FLEET_VAR_HOST_HARDWARE_SERIAL"
		b, err := json.Marshal(ActivityTypeEditedHostNameTemplate{
			TeamID:       new(uint(1)),
			TeamName:     new("Workstations"),
			NameTemplate: &tmpl,
		})
		require.NoError(t, err)
		require.JSONEq(t, `{
			"team_id": 1,
			"team_name": "Workstations",
			"name_template": "$FLEET_VAR_HOST_HARDWARE_SERIAL"
		}`, string(b))
	})

	t.Run("marshal cleared template is null", func(t *testing.T) {
		b, err := json.Marshal(ActivityTypeEditedHostNameTemplate{
			TeamID:   new(uint(1)),
			TeamName: new("Workstations"),
		})
		require.NoError(t, err)
		require.JSONEq(t, `{
			"team_id": 1,
			"team_name": "Workstations",
			"name_template": null
		}`, string(b))
	})

	t.Run("team_id and team_name are renamed in HTTP responses", func(t *testing.T) {
		// The rename to fleet_id/fleet_name happens at the HTTP layer via the
		// renameto struct tags; assert the tags are wired here.
		typ := reflect.TypeFor[ActivityTypeEditedHostNameTemplate]()
		teamID, _ := typ.FieldByName("TeamID")
		teamName, _ := typ.FieldByName("TeamName")
		require.Equal(t, "fleet_id", teamID.Tag.Get("renameto"))
		require.Equal(t, "fleet_name", teamName.Tag.Get("renameto"))
	})
}
