package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestRolesFromSSOAttributes(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		attributes           []SAMLAttribute
		shouldFail           bool
		expectedSSORolesInfo SSORolesInfo
		expectedCoerced      []string
	}{
		{
			name:                 "nil",
			attributes:           nil,
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name:                 "no-role-attributes",
			attributes:           []SAMLAttribute{},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "unknown-key-should-use-default",
			attributes: []SAMLAttribute{
				{
					Name: "foo",
					Values: []SAMLAttributeValue{
						{Value: "bar"},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "global-only",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: ptr.String("admin"),
			},
		},
		{
			name: "global-and-unknown",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: "foo",
					Values: []SAMLAttributeValue{
						{Value: "bar"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: ptr.String("admin"),
			},
		},
		{
			name: "global-and-team",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "5",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
			},
			shouldFail:           true,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "invalid-team-id",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "foo",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
			},
			shouldFail:           true,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "all-teams",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "2",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: nil,
				Teams: []TeamRole{
					{
						ID:   1,
						Role: "observer",
					},
					{
						ID:   2,
						Role: "admin",
					},
				},
			},
		},
		{
			name: "teams-and-unknown",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "2",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: "foo",
					Values: []SAMLAttributeValue{
						{Value: "bar"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: nil,
				Teams: []TeamRole{
					{
						ID:   1,
						Role: "observer",
					},
					{
						ID:   2,
						Role: "admin",
					},
				},
			},
		},
		{
			name: "invalid-global-role",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "administrator"},
					},
				},
			},
			shouldFail: true,
		},
		{
			name: "invalid-team-role",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "administrator"},
					},
				},
			},
			shouldFail: true,
		},
		{
			name: "duplicate-teams",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: "foo",
					Values: []SAMLAttributeValue{
						{Value: "bar"},
					},
				},
			},
			shouldFail: true,
		},
		{
			name: "multi-value-attributes-uses-last",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
						{Value: "admin"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: nil,
				Teams: []TeamRole{
					{
						ID:   1,
						Role: "admin",
					},
				},
			},
		},
		{
			name: "null-value-on-team-attribute-is-ignored",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "null-attributes-on-global-and-team-are-ignored",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "2",
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "null-attributes-are-ignored-should-use-the-set-global-attribute",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: ptr.String("admin"),
			},
		},
		{
			name: "attribute-with-no-values-is-ignored",
			attributes: []SAMLAttribute{
				{
					Name:   globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
			expectedCoerced:      []string{globalUserRoleSSOAttrName},
		},
		{
			name: "empty-string-on-global-is-ignored",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: ""},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
			expectedCoerced:      []string{globalUserRoleSSOAttrName},
		},
		{
			name: "whitespace-only-on-team-is-ignored",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "   "},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
			expectedCoerced:      []string{teamUserRoleSSOAttrNamePrefix + "1"},
		},
		{
			name: "empty-team-attribute-alongside-set-global-attribute",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name:   teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{},
				},
				{
					Name: teamUserRoleSSOAttrNamePrefix + "2",
					Values: []SAMLAttributeValue{
						{Value: ""},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: ptr.String("admin"),
			},
			expectedCoerced: []string{
				teamUserRoleSSOAttrNamePrefix + "1",
				teamUserRoleSSOAttrNamePrefix + "2",
			},
		},
		{
			name: "v2-prefix-empty-value-ignored",
			attributes: []SAMLAttribute{
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_1",
					Values: []SAMLAttributeValue{
						{Value: ""},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
			expectedCoerced:      []string{"FLEET_JIT_USER_ROLE_FLEET_1"},
		},
		{
			// Locks in "last value wins" semantics even when the last value is
			// empty: the attribute is ignored rather than falling back to
			// earlier non-empty values.
			name: "multi-value-last-empty-ignored",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
						{Value: ""},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
			expectedCoerced:      []string{teamUserRoleSSOAttrNamePrefix + "1"},
		},
		{
			// Whitespace around a valid role is NOT trimmed for validation;
			// the value is rejected. Only entirely empty/whitespace-only
			// values are coerced to the null sentinel.
			name: "whitespace-padded-valid-role-fails",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: " admin "},
					},
				},
			},
			shouldFail: true,
		},
		{
			name: "global-technician",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "technician"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: ptr.String("technician"),
			},
		},
		{
			name: "team-technician",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "3",
					Values: []SAMLAttributeValue{
						{Value: "technician"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Teams: []TeamRole{
					{
						ID:   3,
						Role: "technician",
					},
				},
			},
		},
		{
			name: "v2-prefix-all-teams",
			attributes: []SAMLAttribute{
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_2",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: nil,
				Teams: []TeamRole{
					{
						ID:   1,
						Role: "observer",
					},
					{
						ID:   2,
						Role: "admin",
					},
				},
			},
		},
		{
			name: "v2-prefix-global-and-team",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_5",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
			},
			shouldFail:           true,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "v2-prefix-invalid-team-id",
			attributes: []SAMLAttribute{
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_foo",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
			},
			shouldFail:           true,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "v2-prefix-null-value-ignored",
			attributes: []SAMLAttribute{
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_1",
					Values: []SAMLAttributeValue{
						{Value: "null"},
					},
				},
			},
			shouldFail:           false,
			expectedSSORolesInfo: SSORolesInfo{},
		},
		{
			name: "v2-prefix-team-technician",
			attributes: []SAMLAttribute{
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_3",
					Values: []SAMLAttributeValue{
						{Value: "technician"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Teams: []TeamRole{
					{
						ID:   3,
						Role: "technician",
					},
				},
			},
		},
		{
			name: "mixed-v1-and-v2-prefixes",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "observer"},
					},
				},
				{
					Name: "FLEET_JIT_USER_ROLE_FLEET_2",
					Values: []SAMLAttributeValue{
						{Value: "admin"},
					},
				},
			},
			shouldFail: false,
			expectedSSORolesInfo: SSORolesInfo{
				Global: nil,
				Teams: []TeamRole{
					{
						ID:   1,
						Role: "observer",
					},
					{
						ID:   2,
						Role: "admin",
					},
				},
			},
		},
		{
			name: "global-gitops-not-supported-for-jit",
			attributes: []SAMLAttribute{
				{
					Name: globalUserRoleSSOAttrName,
					Values: []SAMLAttributeValue{
						{Value: "gitops"},
					},
				},
			},
			shouldFail: true,
		},
		{
			name: "team-gitops-not-supported-for-jit",
			attributes: []SAMLAttribute{
				{
					Name: teamUserRoleSSOAttrNamePrefix + "1",
					Values: []SAMLAttributeValue{
						{Value: "gitops"},
					},
				},
			},
			shouldFail: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ssoRolesInfo, coerced, err := RolesFromSSOAttributes(tc.attributes)
			if tc.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectedSSORolesInfo, ssoRolesInfo)
			require.Equal(t, tc.expectedCoerced, coerced)
		})
	}
}
