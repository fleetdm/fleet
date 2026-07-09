package fleet

import (
	"encoding/json"
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
		{name: "rtl override format char", tmpl: "bad\u202ename", wantErr: "control characters"},
		{name: "zero-width joiner format char", tmpl: "bad\u200dname", wantErr: "control characters"},
		{name: "invalid utf-8", tmpl: "bad\xff\xfename", wantErr: "valid UTF-8"},
		// The 255-char cap is on the whole template string and counts runes, not
		// bytes; it fires before the byte-floor check below.
		{name: "template over 255 chars", tmpl: strings.Repeat("a", 256), wantErr: "255 characters"},
		{name: "256 multi-byte runes too long", tmpl: strings.Repeat("é", 256), wantErr: "255 characters"},
		// A resolved name can't exceed the device-name byte limit, so a template
		// whose fixed text alone already exceeds it is rejected at save time.
		{name: "literal at 63-byte limit ok", tmpl: strings.Repeat("a", 63), wantNorm: strings.Repeat("a", 63)},
		{name: "literal over 63 bytes", tmpl: strings.Repeat("a", 64), wantErr: "63 bytes"},
		// The floor is bytes, not runes: 32 two-byte runes is 64 bytes.
		{name: "multi-byte literal over 63 bytes", tmpl: strings.Repeat("é", 32), wantErr: "63 bytes"},
		// Only the fixed text counts toward the floor — a short literal plus a
		// (longer) variable token is fine.
		{name: "short literal with variable ok", tmpl: "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL", wantNorm: "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"},
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
		{name: "platform darwin maps to macOS", tmpl: "$FLEET_VAR_HOST_PLATFORM", host: host, want: "macOS"},
		{
			name: "platform ios maps to iOS",
			tmpl: "$FLEET_VAR_HOST_PLATFORM",
			host: &Host{Platform: "ios"},
			want: "iOS",
		},
		{
			name: "platform ipados maps to iPadOS",
			tmpl: "$FLEET_VAR_HOST_PLATFORM",
			host: &Host{Platform: "ipados"},
			want: "iPadOS",
		},
		{
			name: "mixed and repeated",
			tmpl: "$FLEET_VAR_HOST_PLATFORM-$FLEET_VAR_HOST_HARDWARE_SERIAL-${FLEET_VAR_HOST_HARDWARE_SERIAL}",
			host: host,
			want: "macOS-C02ABC123-C02ABC123",
		},
		{
			// Non-Apple platforms fall back to the raw value (the feature only
			// applies to Apple devices, so this is defensive).
			name: "non-apple platform falls back to raw value",
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

func TestTeamHostNameTemplateRoundTrip(t *testing.T) {
	t.Run("TeamMDM", func(t *testing.T) {
		in := TeamMDM{HostNameTemplate: "$FLEET_VAR_HOST_HARDWARE_SERIAL"}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_HARDWARE_SERIAL"`)

		var out TeamMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.Equal(t, in.HostNameTemplate, out.HostNameTemplate)
	})

	t.Run("TeamSpecMDM", func(t *testing.T) {
		in := TeamSpecMDM{HostNameTemplate: optjson.SetString("$FLEET_VAR_HOST_UUID")}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_UUID"`)

		var out TeamSpecMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.True(t, out.HostNameTemplate.Set)
		require.True(t, out.HostNameTemplate.Valid)
		require.Equal(t, "$FLEET_VAR_HOST_UUID", out.HostNameTemplate.Value)
	})

	t.Run("TeamSpecMDM absent", func(t *testing.T) {
		var out TeamSpecMDM
		require.NoError(t, json.Unmarshal([]byte(`{}`), &out))
		require.False(t, out.HostNameTemplate.Set)
	})

	t.Run("TeamPayloadMDM", func(t *testing.T) {
		in := TeamPayloadMDM{HostNameTemplate: optjson.SetString("$FLEET_VAR_HOST_PLATFORM")}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(b), `"name_template":"$FLEET_VAR_HOST_PLATFORM"`)

		var out TeamPayloadMDM
		require.NoError(t, json.Unmarshal(b, &out))
		require.True(t, out.HostNameTemplate.Set)
		require.Equal(t, "$FLEET_VAR_HOST_PLATFORM", out.HostNameTemplate.Value)
	})

	t.Run("TeamConfig storage round-trip", func(t *testing.T) {
		// name_template rides in the teams.config JSON blob (no dedicated
		// column), so verify it survives the actual SQL Value()/Scan() boundary.
		in := TeamConfig{MDM: TeamMDM{HostNameTemplate: "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}
		val, err := in.Value()
		require.NoError(t, err)

		var out TeamConfig
		require.NoError(t, out.Scan(val))
		require.Equal(t, "$FLEET_VAR_HOST_HARDWARE_SERIAL", out.MDM.HostNameTemplate)
	})
}

func TestActivityTypeEditedHostNameTemplate(t *testing.T) {
	require.Equal(t, "edited_host_name_template", ActivityTypeEditedHostNameTemplate{}.ActivityName())

	t.Run("marshal with template", func(t *testing.T) {
		tmpl := "$FLEET_VAR_HOST_HARDWARE_SERIAL"
		b, err := json.Marshal(ActivityTypeEditedHostNameTemplate{
			FleetID:          new(uint(1)),
			FleetName:        new("Workstations"),
			HostNameTemplate: &tmpl,
		})
		require.NoError(t, err)
		require.JSONEq(t, `{
			"fleet_id": 1,
			"fleet_name": "Workstations",
			"name_template": "$FLEET_VAR_HOST_HARDWARE_SERIAL"
		}`, string(b))
	})

	t.Run("marshal cleared template is null", func(t *testing.T) {
		b, err := json.Marshal(ActivityTypeEditedHostNameTemplate{
			FleetID:   new(uint(1)),
			FleetName: new("Workstations"),
		})
		require.NoError(t, err)
		require.JSONEq(t, `{
			"fleet_id": 1,
			"fleet_name": "Workstations",
			"name_template": null
		}`, string(b))
	})
}
