package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstituteFleetVarsAndVitalsInAndroidAppConfig(t *testing.T) {
	ctx := t.Context()
	host := AndroidAppConfigSubstitutionHost{
		HostID:         42,
		UUID:           "host-uuid-1",
		HardwareSerial: "ABC123",
		Platform:       "android",
	}
	emptyDS := new(mock.Store)

	t.Run("nil config returns nil", func(t *testing.T) {
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, nil, host)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("empty config returns empty", func(t *testing.T) {
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, []byte{}, host)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("config without variables returns unchanged", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"key": "plain"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Equal(t, cfg, got)
	})

	t.Run("HOST_UUID substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"deviceId": "$FLEET_VAR_HOST_UUID"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "host-uuid-1")
		require.NotContains(t, string(got), "$FLEET_VAR_HOST_UUID")
		// Verify it's still valid JSON
		require.True(t, json.Valid(got))
	})

	t.Run("HOST_UUID with braces substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"deviceId": "${FLEET_VAR_HOST_UUID}"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "host-uuid-1")
		require.NotContains(t, string(got), "${FLEET_VAR_HOST_UUID}")
	})

	t.Run("HOST_HARDWARE_SERIAL substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "ABC123")
	})

	t.Run("HOST_HARDWARE_SERIAL empty returns error", func(t *testing.T) {
		noSerialHost := host
		noSerialHost.HardwareSerial = ""
		cfg := []byte(`{"managedConfiguration": {"serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, noSerialHost)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("HOST_PLATFORM substituted", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"platform": "$FLEET_VAR_HOST_PLATFORM"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("multiple variables substituted independently", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return []string{"user@example.com"}, nil
		}
		cfg := []byte(`{"managedConfiguration": {"uuid": "$FLEET_VAR_HOST_UUID", "serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL", "email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "jdoe")
		require.NotContains(t, string(got), "@example.com")
		require.True(t, json.Valid(got))
	})

	t.Run("IDP external ID substituted", func(t *testing.T) {
		ds := new(mock.Store)
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
			return []uint{42}, nil
		}
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "jdoe@example.com", ExternalID: new("00u1abcd2EFGH3ijk4x7")}, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		cfg := []byte(`{"managedConfiguration": {"id": "$FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.JSONEq(t, `{"managedConfiguration": {"id": "00u1abcd2EFGH3ijk4x7"}}`, string(got))
	})

	t.Run("IDP external ID missing returns error", func(t *testing.T) {
		ds := new(mock.Store)
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
			return []uint{42}, nil
		}
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "jdoe@example.com"}, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		cfg := []byte(`{"managedConfiguration": {"id": "$FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
		require.ErrorIs(t, err, ErrUnresolvableAndroidAppConfigVar)
		require.ErrorContains(t, err, "There is no IdP external ID for this host.")
		require.Nil(t, got)
	})

	t.Run("unsupported variable returns error", func(t *testing.T) {
		cfg := []byte(`{"managedConfiguration": {"chal": "$FLEET_VAR_NDES_SCEP_CHALLENGE"}}`)
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, emptyDS, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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
		got, err := SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, ds, cfg, host)
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

// A host that can't supply a referenced value must not take the rest of its
// batch down with it: the other hosts still get their app config, and the job
// succeeds rather than burning its retries.
func TestIsUnresolvableAndroidAppConfigForHost(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"missing vital value", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{1}}, true},
		{"wrapped missing vital value", fmt.Errorf("substitute: %w", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{1}}), true},
		{"unresolvable fleet var", &UnresolvableAndroidAppConfigVarError{FleetVar: "HOST_END_USER_IDP_USERNAME"}, true},
		{"wrapped unresolvable fleet var", fmt.Errorf("substitute: %w", &UnresolvableAndroidAppConfigVarError{FleetVar: "HOST_END_USER_IDP_USERNAME"}), true},
		{"unrelated error", errors.New("db is down"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, IsUnresolvableAndroidAppConfigForHost(c.err))
		})
	}
}

func TestBuildApplicationPoliciesWithConfig(t *testing.T) {
	ctx := t.Context()

	t.Run("app with full configuration", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"managedConfiguration": {"key": "value"}, "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED", "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
		}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "com.example.app", policies[0].PackageName)
		require.Equal(t, "AVAILABLE", policies[0].InstallType)
		require.JSONEq(t, `{"key": "value"}`, string(policies[0].ManagedConfiguration))
		require.Equal(t, "WORK_PROFILE_WIDGETS_ALLOWED", policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_ALLOWED", policies[0].CredentialProviderPolicy)
	})

	t.Run("app without configuration clears previously applied settings", func(t *testing.T) {
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, nil, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Empty(t, policies[0].ManagedConfiguration)
		require.Equal(t, "WORK_PROFILE_WIDGETS_UNSPECIFIED", policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED", policies[0].CredentialProviderPolicy)
	})

	t.Run("partial configuration leaves other fields unset", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
		}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, "FORCE_INSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "FORCE_INSTALLED", policies[0].InstallType)
		require.Empty(t, policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_ALLOWED", policies[0].CredentialProviderPolicy)
	})

	t.Run("empty configuration object leaves every field unset", func(t *testing.T) {
		configs := map[string][]byte{"com.example.app": []byte(`{}`)}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Nil(t, policies[0].ManagedConfiguration)
		require.Empty(t, policies[0].WorkProfileWidgets)
		require.Empty(t, policies[0].CredentialProviderPolicy)
	})

	t.Run("other ApplicationPolicy fields are passed through", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"autoUpdateMode": "AUTO_UPDATE_HIGH_PRIORITY", "minimumVersionCode": 42, "permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "GRANT"}]}`),
		}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, "AUTO_UPDATE_HIGH_PRIORITY", policies[0].AutoUpdateMode)
		assert.EqualValues(t, 42, policies[0].MinimumVersionCode)
		require.Len(t, policies[0].PermissionGrants, 1)
		assert.Equal(t, "android.permission.CAMERA", policies[0].PermissionGrants[0].Permission)
		assert.Equal(t, "GRANT", policies[0].PermissionGrants[0].Policy)
	})

	t.Run("stored packageName and installType never override Fleet's values", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"packageName": "com.example.other", "installType": "KIOSK"}`),
		}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, "PREINSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, "com.example.app", policies[0].PackageName)
		assert.Equal(t, "PREINSTALLED", policies[0].InstallType)
	})

	t.Run("installConstraint is sent only for installs Google starts", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"installConstraint": [{"chargingConstraint": "INSTALL_ONLY_WHEN_CHARGING"}], "autoUpdateMode": "AUTO_UPDATE_POSTPONED"}`),
		}
		cases := []struct {
			installType    string
			wantConstraint bool
		}{
			{installType: "AVAILABLE", wantConstraint: false},
			{installType: "PREINSTALLED", wantConstraint: true},
			{installType: "FORCE_INSTALLED", wantConstraint: true},
		}
		for _, c := range cases {
			t.Run(c.installType, func(t *testing.T) {
				policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.app"}, configs, c.installType)
				require.NoError(t, err)
				require.Len(t, policies, 1)
				assert.Equal(t, "AUTO_UPDATE_POSTPONED", policies[0].AutoUpdateMode)
				if c.wantConstraint {
					require.Len(t, policies[0].InstallConstraint, 1)
					assert.Equal(t, "INSTALL_ONLY_WHEN_CHARGING", policies[0].InstallConstraint[0].ChargingConstraint)
				} else {
					assert.Empty(t, policies[0].InstallConstraint)
				}
			})
		}
	})

	t.Run("each app gets its own configuration", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.configured": []byte(`{"autoUpdateMode": "AUTO_UPDATE_POSTPONED"}`),
		}
		policies, err := BuildApplicationPoliciesWithConfig(ctx, []string{"com.example.configured", "com.example.unconfigured"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 2)
		assert.Equal(t, "com.example.configured", policies[0].PackageName)
		assert.Equal(t, "AUTO_UPDATE_POSTPONED", policies[0].AutoUpdateMode)
		assert.Empty(t, policies[0].WorkProfileWidgets)
		assert.Equal(t, "com.example.unconfigured", policies[1].PackageName)
		assert.Empty(t, policies[1].AutoUpdateMode)
		assert.Equal(t, "WORK_PROFILE_WIDGETS_UNSPECIFIED", policies[1].WorkProfileWidgets)
	})
}
