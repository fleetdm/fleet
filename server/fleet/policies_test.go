package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPolicyPlatforms(t *testing.T) {
	testCases := []struct {
		platforms string
		isValid   bool
	}{
		{"windows,chrome", true},
		{"chrome", true},
		{"bados", false},
	}

	for _, tc := range testCases {
		actual := verifyPolicyPlatforms(tc.platforms)

		if tc.isValid {
			require.NoError(t, actual)
			continue
		}
		require.Error(t, actual)
	}
}

func TestFirstFuplicatePolicySpecName(t *testing.T) {
	testCases := []struct {
		name     string
		result   string
		policies []*PolicySpec
	}{
		{"no specs", "", []*PolicySpec{}},
		{"no duplicate names", "", []*PolicySpec{{Name: "foo"}}},
		{"duplicate names", "foo", []*PolicySpec{{Name: "foo"}, {Name: "bar"}, {Name: "foo"}}},
	}

	for _, tc := range testCases {
		name := FirstDuplicatePolicySpecName(tc.policies)
		require.Equal(t, tc.result, name)
	}
}

func TestPolicyPayloadVerify_MDMType(t *testing.T) {
	validDef := json.RawMessage(`[{"field":"OSVersion","operator":"version_gte","expected":"17.0","source":"DeviceInformation"}]`)

	testCases := []struct {
		name    string
		payload PolicyPayload
		wantErr string
	}{
		{
			name: "valid MDM policy",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				Name:               "iOS version check",
				Platform:           "ios",
				MDMCheckDefinition: &validDef,
			},
		},
		{
			name: "MDM policy with query_id fails",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				Name:               "test",
				QueryID:            ptrUint(1),
				MDMCheckDefinition: &validDef,
			},
			wantErr: "MDM policies cannot use query_id",
		},
		{
			name: "MDM policy with query fails",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				Name:               "test",
				Query:              "SELECT 1",
				MDMCheckDefinition: &validDef,
			},
			wantErr: "MDM policies cannot have a query",
		},
		{
			name: "MDM policy without name fails",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				MDMCheckDefinition: &validDef,
			},
			wantErr: "policy name cannot be empty",
		},
		{
			name: "MDM policy without definition fails",
			payload: PolicyPayload{
				Type: PolicyTypeMDM,
				Name: "test",
			},
			wantErr: "MDM policies must have mdm_check_definition",
		},
		{
			name: "MDM policy with invalid platform fails",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				Name:               "test",
				Platform:           "bados",
				MDMCheckDefinition: &validDef,
			},
			wantErr: "invalid policy platform",
		},
		{
			name: "MDM policy with conflicting labels fails",
			payload: PolicyPayload{
				Type:               PolicyTypeMDM,
				Name:               "test",
				Platform:           "ios",
				MDMCheckDefinition: &validDef,
				LabelsIncludeAny:   []string{"a"},
				LabelsExcludeAny:   []string{"b"},
			},
			wantErr: "cannot include both",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.payload.Verify()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestPolicySpecVerify_MDMType(t *testing.T) {
	testCases := []struct {
		name    string
		spec    PolicySpec
		wantErr string
	}{
		{
			name: "valid MDM spec",
			spec: PolicySpec{
				Type:     PolicyTypeMDM,
				Name:     "iOS version check",
				Platform: "ios",
				MDMChecks: []MDMPolicyCheck{
					{Field: "OSVersion", Operator: MDMPolicyCheckVersionGte, Expected: "17.0", Source: MDMPolicySourceDeviceInformation},
				},
			},
		},
		{
			name: "MDM spec with query fails",
			spec: PolicySpec{
				Type:  PolicyTypeMDM,
				Name:  "test",
				Query: "SELECT 1",
				MDMChecks: []MDMPolicyCheck{
					{Field: "OSVersion", Operator: MDMPolicyCheckVersionGte, Expected: "17.0", Source: MDMPolicySourceDeviceInformation},
				},
			},
			wantErr: "MDM policies cannot have a query",
		},
		{
			name: "MDM spec without checks fails",
			spec: PolicySpec{
				Type: PolicyTypeMDM,
				Name: "test",
			},
			wantErr: "MDM policies must have mdm_checks",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Verify()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func ptrUint(v uint) *uint {
	return &v
}
