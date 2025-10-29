//go:build darwin
// +build darwin

package app_sso_platform

import (
	"context"
	_ "embed"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/app_sso_platform_state_sample1.txt
	sample1 string

	//go:embed testdata/app_sso_platform_state_sample2_user_null.txt
	sample2 string

	//go:embed testdata/app_sso_platform_state_empty.txt
	empty string
)

func TestParseAppSSOPlatformCommandOutput(t *testing.T) {
	// Match
	data, err := parseAppSSOPlatformCommandOutput([]byte(sample1), "com.microsoft.CompanyPortalMac.ssoextension", "KERBEROS.MICROSOFTONLINE.COM")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, "34b1ba9a-3b2d-4c6c-ab4b-615f4b143eab", data.deviceID)
	require.Equal(t, "com.microsoft.CompanyPortalMac.ssoextension", data.extensionIdentifier)
	require.Equal(t, "KERBEROS.MICROSOFTONLINE.COM", data.realm)
	require.Equal(t, "foobar@contoso.onmicrosoft.com", data.userPrincipalName)

	// Empty, Platform SSO not set yet.
	data, err = parseAppSSOPlatformCommandOutput([]byte(empty), "com.microsoft.CompanyPortalMac.ssoextension", "KERBEROS.MICROSOFTONLINE.COM")
	require.NoError(t, err)
	require.Nil(t, data)

	// Platform SSO extension identifier does not match.
	data, err = parseAppSSOPlatformCommandOutput([]byte(sample1), "com.microsoft.Other.other", "KERBEROS.MICROSOFTONLINE.COM")
	require.NoError(t, err)
	require.Nil(t, data)

	// Platform SSO extension identifier matches, but user realm doesn't match.
	data, err = parseAppSSOPlatformCommandOutput([]byte(sample1), "com.microsoft.CompanyPortalMac.ssoextension", "FOOBAR.OTHER.COM")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, "34b1ba9a-3b2d-4c6c-ab4b-615f4b143eab", data.deviceID)
	require.Equal(t, "com.microsoft.CompanyPortalMac.ssoextension", data.extensionIdentifier)
	require.Equal(t, "FOOBAR.OTHER.COM", data.realm)
	require.Equal(t, "", data.userPrincipalName)

	// None matches.
	data, err = parseAppSSOPlatformCommandOutput([]byte(sample1), "com.microsoft.Other.other", "FOOBAR.OTHER.COM")
	require.NoError(t, err)
	require.Nil(t, data)

	// Platform SSO extension identifier matches, but user is not registered yet (null).
	// Can happen if Platform SSO configuration profile was deployed and this is a workstation with two users,
	// and one user registered but not the other one.
	data, err = parseAppSSOPlatformCommandOutput([]byte(sample2), "com.microsoft.CompanyPortalMac.ssoextension", "KERBEROS.MICROSOFTONLINE.COM")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, "34b1ba9a-3b2d-4c6c-ab4b-615f4b143eab", data.deviceID)
	require.Equal(t, "com.microsoft.CompanyPortalMac.ssoextension", data.extensionIdentifier)
	require.Equal(t, "KERBEROS.MICROSOFTONLINE.COM", data.realm)
	require.Equal(t, "", data.userPrincipalName)
}

func TestGenerateErrors(t *testing.T) {
	// Multiple extension_identifier values.
	_, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value1",
					},
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value2",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Multiple realm values.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value1",
					},
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value2",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Multiple extension_identifier value.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Missing realm value.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Not using equality on extension_identifier.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorLike,
						Expression: "extension_identifier_value",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Not using equality on realm.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorLike,
						Expression: "realm_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Empty extension_identifier.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "realm_value",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Empty realm.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "extension_identifier_value",
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "",
					},
				},
			},
		},
	})
	require.Error(t, err)
}
