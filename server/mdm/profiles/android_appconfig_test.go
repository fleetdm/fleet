package profiles

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestSubstituteFleetVarsInAndroidAppConfig(t *testing.T) {
	ctx := t.Context()
	host := AndroidAppConfigSubstitutionHost{
		HostID:         42,
		UUID:           "host-uuid-1",
		HardwareSerial: "ABC123",
		Platform:       "android",
	}
	emptyDS := new(mock.Store)

	t.Run("nil config returns nil", func(t *testing.T) {
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, nil, host)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("empty config returns empty", func(t *testing.T) {
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, []byte{}, host)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("config without variables returns unchanged", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"key": "plain"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Equal(t, cfg, got)
	})

	t.Run("HOST_UUID substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"deviceId": "$FLEET_VAR_HOST_UUID"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "host-uuid-1")
		require.NotContains(t, string(got), "$FLEET_VAR_HOST_UUID")
		// Verify it's still valid JSON
		require.True(t, json.Valid(got))
	})

	t.Run("HOST_UUID with braces substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"deviceId": "${FLEET_VAR_HOST_UUID}"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "host-uuid-1")
		require.NotContains(t, string(got), "${FLEET_VAR_HOST_UUID}")
	})

	t.Run("HOST_HARDWARE_SERIAL substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "ABC123")
	})

	t.Run("HOST_HARDWARE_SERIAL empty returns error", func(t *testing.T) {
		noSerialHost := host
		noSerialHost.HardwareSerial = ""
		cfg := []byte(`{"managedConfiguration": {"serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, noSerialHost)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("HOST_PLATFORM substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"platform": "$FLEET_VAR_HOST_PLATFORM"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "android")
	})

	t.Run("HOST_END_USER_EMAIL_IDP resolves via datastore", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			require.Equal(t, "host-uuid-1", hostUUID)
			return []string{"user@example.com"}, nil
		}
		cfg := []byte(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "user@example.com")
		require.True(t, json.Valid(got))
	})

	t.Run("HOST_END_USER_EMAIL_IDP missing returns error", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return nil, nil
		}
		cfg := []byte(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("multiple variables substituted independently", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return []string{"user@example.com"}, nil
		}
		cfg := []byte(`{"managedConfiguration": {"uuid": "$FLEET_VAR_HOST_UUID", "serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL", "email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		s := string(got)
		require.Contains(t, s, "host-uuid-1")
		require.Contains(t, s, "ABC123")
		require.Contains(t, s, "user@example.com")
		require.True(t, json.Valid(got))
	})

	t.Run("JSON special chars in value are escaped", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return []string{`user"with\special`}, nil
		}
		cfg := []byte(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.True(t, json.Valid(got), "result must be valid JSON: %s", string(got))
		// Parse and verify the value round-trips correctly
		var parsed map[string]map[string]string
		require.NoError(t, json.Unmarshal(got, &parsed))
		require.Equal(t, `user"with\special`, parsed["managedConfiguration"]["email"])
	})

	t.Run("IDP username substituted", func(t *testing.T) {
		ds := new(mock.Store)
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
			return []uint{42}, nil
		}
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "jdoe@example.com", GivenName: new("John"), FamilyName: new("Doe")}, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		cfg := []byte(`{"managedConfiguration": {"user": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "jdoe@example.com")
		require.True(t, json.Valid(got))
	})

	t.Run("IDP username local part substituted", func(t *testing.T) {
		ds := new(mock.Store)
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
			return []uint{42}, nil
		}
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "jdoe@example.com", GivenName: new("John"), FamilyName: new("Doe")}, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		cfg := []byte(`{"managedConfiguration": {"user": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "jdoe")
		require.NotContains(t, string(got), "@example.com")
		require.True(t, json.Valid(got))
	})

	t.Run("unsupported variable returns error", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"chal": "$FLEET_VAR_NDES_SCEP_CHALLENGE"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("custom host vital substituted", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
			require.EqualValues(t, 42, hostID)
			require.Contains(t, document, "$FLEET_HOST_VITAL_7")
			return `{"managedConfiguration": {"assetTag": "asset-123"}}`, nil
		}
		cfg := []byte(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_7"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "asset-123")
		require.True(t, ds.ExpandCustomHostVitalsFuncInvoked)
	})

	t.Run("custom host vital alongside a Fleet variable, both substituted", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
			// Called after the $FLEET_VAR_ substitution above has already run, so
			// the document should carry the resolved UUID, not the token.
			require.Contains(t, document, "host-uuid-1")
			return strings.ReplaceAll(document, "$FLEET_HOST_VITAL_7", "asset-123"), nil
		}
		cfg := []byte(`{"managedConfiguration": {"uuid": "$FLEET_VAR_HOST_UUID", "assetTag": "$FLEET_HOST_VITAL_7"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		s := string(got)
		require.Contains(t, s, "host-uuid-1")
		require.Contains(t, s, "asset-123")
	})

	t.Run("custom host vital with no value set for host returns error", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
			return "", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{7}}
		}
		cfg := []byte(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_7"}}`)
		got, err := SubstituteFleetVarsInAndroidAppConfig(ctx, ds, cfg, host)
		var missing *fleet.MissingCustomHostVitalValueError
		require.ErrorAs(t, err, &missing)
		require.Nil(t, got)
	})
}

func TestContainsFleetVarOrCustomHostVital(t *testing.T) {
	require.True(t, ContainsFleetVarOrCustomHostVital([]byte(`{"a": "$FLEET_VAR_HOST_UUID"}`)))
	require.True(t, ContainsFleetVarOrCustomHostVital([]byte(`{"a": "$FLEET_HOST_VITAL_7"}`)))
	require.False(t, ContainsFleetVarOrCustomHostVital([]byte(`{"a": "plain"}`)))
	require.False(t, ContainsFleetVarOrCustomHostVital([]byte(`{"a": "FLEET_HOST_VITAL_no_dollar_sign"}`)))
}

func TestJsonEscapeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain", "plain"},
		{`has "quotes"`, `has \"quotes\"`},
		{"has\\backslash", "has\\\\backslash"},
		{"has\nnewline", "has\\nnewline"},
		{"has\ttab", "has\\ttab"},
		{"normal-uuid-123", "normal-uuid-123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, jsonEscapeString(tt.input))
		})
	}
}
