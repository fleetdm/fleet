package apple_mdm

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleEnrollURL(t *testing.T) {
	cases := []struct {
		appConfig   *fleet.AppConfig
		expectedURL string
	}{
		{
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://foo.example.com",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
		{
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://foo.example.com/",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
	}

	for _, tt := range cases {
		enrollURL, err := EnrollURL("tok", tt.appConfig)
		require.NoError(t, err)
		require.Equal(t, tt.expectedURL, enrollURL)
	}
}

func TestGenerateRandomPin(t *testing.T) {
	for i := 1; i <= 100; i++ {
		pin, err := GenerateRandomPin(i)
		require.NoError(t, err)
		require.Len(t, pin, i)
	}
}

func TestIsRecoveryLockPasswordMismatchError(t *testing.T) {
	cases := []struct {
		name     string
		chain    []mdm.ErrorChain
		expected bool
	}{
		{
			name:     "empty chain",
			chain:    nil,
			expected: false,
		},
		{
			name: "MDMClientError 70 - existing password not provided",
			chain: []mdm.ErrorChain{
				{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Existing recovery lock password not provided"},
			},
			expected: true,
		},
		{
			name: "ROSLockoutServiceDaemonErrorDomain 8 - password failed to validate",
			chain: []mdm.ErrorChain{
				{ErrorCode: 8, ErrorDomain: "ROSLockoutServiceDaemonErrorDomain", LocalizedDescription: "The provided recovery password failed to validate."},
			},
			expected: true,
		},
		{
			name: "different MDMClientError code",
			chain: []mdm.ErrorChain{
				{ErrorCode: 71, ErrorDomain: "MDMClientError", LocalizedDescription: "Some other error"},
			},
			expected: false,
		},
		{
			name: "different error domain",
			chain: []mdm.ErrorChain{
				{ErrorCode: 70, ErrorDomain: "SomeOtherDomain", LocalizedDescription: "Some error"},
			},
			expected: false,
		},
		{
			name: "generic transient error",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12345, ErrorDomain: "test", LocalizedDescription: "Network timeout"},
			},
			expected: false,
		},
		{
			name: "password mismatch in chain with other errors",
			chain: []mdm.ErrorChain{
				{ErrorCode: 100, ErrorDomain: "SomeOtherDomain", LocalizedDescription: "First error"},
				{ErrorCode: 8, ErrorDomain: "ROSLockoutServiceDaemonErrorDomain", LocalizedDescription: "The provided recovery password failed to validate."},
			},
			expected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoveryLockPasswordMismatchError(tt.chain)
			require.Equal(t, tt.expected, result)
		})
	}
}
