package apple_mdm

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
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

func TestIsProfileNotFoundError(t *testing.T) {
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
			name: "MDMClientError 89 - profile not found",
			chain: []mdm.ErrorChain{
				{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'com.example' not found."},
			},
			expected: true,
		},
		{
			name: "different MDMClientError code",
			chain: []mdm.ErrorChain{
				{ErrorCode: 90, ErrorDomain: "MDMClientError", USEnglishDescription: "Some other error"},
			},
			expected: false,
		},
		{
			name: "different error domain with code 89",
			chain: []mdm.ErrorChain{
				{ErrorCode: 89, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "Some error"},
			},
			expected: false,
		},
		{
			name: "profile not found in chain with other errors",
			chain: []mdm.ErrorChain{
				{ErrorCode: 100, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "First error"},
				{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'com.example' not found."},
			},
			expected: true,
		},
		{
			name: "MDMErrorDomain 12075 - profile not installed",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12075, ErrorDomain: "MDMErrorDomain", USEnglishDescription: "The profile 'com.example' is not installed."},
			},
			expected: true,
		},
		{
			name: "different MDMErrorDomain code",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12076, ErrorDomain: "MDMErrorDomain", USEnglishDescription: "Some other error"},
			},
			expected: false,
		},
		{
			name: "different error domain with code 12075",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12075, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "Some error"},
			},
			expected: false,
		},
		{
			name: "profile not installed in chain with other errors",
			chain: []mdm.ErrorChain{
				{ErrorCode: 100, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "First error"},
				{ErrorCode: 12075, ErrorDomain: "MDMErrorDomain", USEnglishDescription: "The profile 'com.example' is not installed."},
			},
			expected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProfileNotFoundError(tt.chain)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAppAlreadyInstalledError(t *testing.T) {
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
			name: "MCMDMErrorDomain 12042 - canonical AlreadyInstalled",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12042, ErrorDomain: "MCMDMErrorDomain", USEnglishDescription: "The app with iTunes Store ID 546505307 is already installed."},
			},
			expected: true,
		},
		{
			name: "matched by message even with unknown code",
			chain: []mdm.ErrorChain{
				{ErrorCode: 99999, ErrorDomain: "SomeFutureDomain", USEnglishDescription: "The app with iTunes Store ID 546505307 is already installed."},
			},
			expected: true,
		},
		{
			name: "matched by LocalizedDescription when USEnglishDescription is empty",
			chain: []mdm.ErrorChain{
				{ErrorCode: 99999, ErrorDomain: "SomeFutureDomain", LocalizedDescription: "The app with iTunes Store ID 546505307 is already installed."},
			},
			expected: true,
		},
		{
			name: "matched somewhere in chain",
			chain: []mdm.ErrorChain{
				{ErrorCode: 100, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "First error"},
				{ErrorCode: 12042, ErrorDomain: "MCMDMErrorDomain", USEnglishDescription: "The app with iTunes Store ID 1 is already installed."},
			},
			expected: true,
		},
		{
			name: "unrelated error",
			chain: []mdm.ErrorChain{
				{ErrorCode: 9610, ErrorDomain: "MCMDMErrorDomain", USEnglishDescription: "Cannot establish a connection."},
			},
			expected: false,
		},
		{
			name: "different domain with code 12042",
			chain: []mdm.ErrorChain{
				{ErrorCode: 12042, ErrorDomain: "SomeOtherDomain", USEnglishDescription: "Resource Already Exists"},
			},
			expected: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, IsAppAlreadyInstalledError(tt.chain))
		})
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

func TestGenerateManagedAccountPassword(t *testing.T) {
	pw := GenerateManagedAccountPassword()

	// Format: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX (6 groups of 4 chars separated by dashes)
	groups := strings.Split(pw, "-")
	require.Len(t, groups, ManagedAccountPasswordGroupCount)
	for _, g := range groups {
		require.Len(t, g, ManagedAccountPasswordGroupLen)
		for _, c := range g {
			assert.Contains(t, RecoveryLockPasswordCharset, string(c))
		}
	}

	// Two calls should produce different passwords (with overwhelming probability).
	pw2 := GenerateManagedAccountPassword()
	require.NotEqual(t, pw, pw2)
}

func TestGenerateSaltedSHA512PBKDF2Hash(t *testing.T) {
	data, err := GenerateSaltedSHA512PBKDF2Hash("test-password")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Parse the plist and verify the structure.
	var result saltedSHA512PBKDF2
	_, err = plist.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Len(t, result.PBKDF2.Salt, pbkdf2SaltLen, "salt should be %d bytes", pbkdf2SaltLen)
	assert.Len(t, result.PBKDF2.Entropy, pbkdf2KeyLen, "entropy should be %d bytes", pbkdf2KeyLen)
	assert.Equal(t, pbkdf2Iterations, result.PBKDF2.Iterations)

	// Two calls with the same password should produce different outputs (different random salts).
	data2, err := GenerateSaltedSHA512PBKDF2Hash("test-password")
	require.NoError(t, err)
	require.NotEqual(t, data, data2)
}
