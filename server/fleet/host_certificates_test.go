package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractHostCertificateNameDetails(t *testing.T) {
	expected := HostCertificateNameDetails{
		Country:            "US",
		Organization:       "Fleet Device Management Inc.",
		OrganizationalUnit: "Fleet Device Management Inc.",
		CommonName:         "FleetDM",
	}

	cases := []struct {
		name     string
		input    string
		expected *HostCertificateNameDetails
		err      bool
	}{
		{
			name:     "valid",
			input:    "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM",
			expected: &expected,
		},
		{
			name:     "valid with different order",
			input:    "/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/C=US",
			expected: &expected,
		},
		{
			name:  "valid with missing key",
			input: "/C=US/O=Fleet Device Management Inc./CN=FleetDM ",
			expected: &HostCertificateNameDetails{
				Country:            "US",
				Organization:       "Fleet Device Management Inc.",
				OrganizationalUnit: "",
				CommonName:         "FleetDM",
			},
		},
		{
			name:     "valid with additional keyr",
			input:    "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/L=SomeCity",
			expected: &expected,
		},
		{
			name:  "invalid format with extra slash",
			input: "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/invalid",
			err:   true,
		},
		{
			name:  "invalid format with wrong separator",
			input: "C=US,O=Fleet Device Management Inc.,OU=Fleet Device Management Inc.,CN=FleetDM",
			err:   true,
		},
		{
			name:  "invalid format with extra equal",
			input: "/C=US=/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM",
			err:   true,
		},
		{
			name:  "invalid format with malformed key values",
			input: "/C=US/O/OU=Fleet Device Management Inc./=/CN=FleetDM",
			err:   true,
		},
		{
			name:  "empty",
			input: "",
			err:   true,
		},
		{
			name:  "missing value",
			input: "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=",
			expected: &HostCertificateNameDetails{
				Country:            "US",
				Organization:       "Fleet Device Management Inc.",
				OrganizationalUnit: "Fleet Device Management Inc.",
				CommonName:         "",
			},
		},
		{
			name:     "missing first slash",
			input:    "C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM",
			expected: &expected,
		},
		{
			name:     "trailing slash",
			input:    "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/",
			expected: &expected,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ExtractDetailsFromOsqueryDistinguishedName(tc.input)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}
