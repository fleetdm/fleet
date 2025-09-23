package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractHostCertificateNameDetails(t *testing.T) {
	getExpectedHostCertificateDetails := func(commonName string) *HostCertificateNameDetails {
		return &HostCertificateNameDetails{
			Country:            "US",
			Organization:       "Fleet Device Management Inc.",
			OrganizationalUnit: "Fleet Device Management Inc.",
			CommonName:         commonName,
		}
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
			expected: getExpectedHostCertificateDetails("FleetDM"),
		},
		{
			name:     "valid with different order",
			input:    "/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/C=US",
			expected: getExpectedHostCertificateDetails("FleetDM"),
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
			expected: getExpectedHostCertificateDetails("FleetDM"),
		},
		{
			name:     "valid format with extra slash",
			input:    `/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM\/valid`,
			expected: getExpectedHostCertificateDetails("FleetDM/valid"),
		},
		{
			name:     "valid with safe escape sequence",
			input:    `/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM<<SLASH>>valid`,
			expected: getExpectedHostCertificateDetails("FleetDM/valid"),
		},
		{
			name:     "valid format with equal signs in value",
			input:    "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM=Company",
			expected: getExpectedHostCertificateDetails("FleetDM=Company"),
		},
		{
			name:  "invalid format with extra slash without escape",
			input: "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/invalid",
			err:   true,
		},
		{
			name:  "format with wrong separator", // this will now just be treated as part of C
			input: "C=US,O=Fleet Device Management Inc.,OU=Fleet Device Management Inc.,CN=FleetDM",
			expected: &HostCertificateNameDetails{
				Country: "US,O=Fleet Device Management Inc.,OU=Fleet Device Management Inc.,CN=FleetDM",
			},
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
			expected: getExpectedHostCertificateDetails("FleetDM"),
		},
		{
			name:     "trailing slash",
			input:    "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM/",
			expected: getExpectedHostCertificateDetails("FleetDM"),
		},
		{
			name:  "simple common name",
			input: "/CN=FleetDM",
			expected: &HostCertificateNameDetails{
				Country:            "",
				Organization:       "",
				OrganizationalUnit: "",
				CommonName:         "FleetDM",
			},
		},
		{
			name:  "simple common name with no leading slash",
			input: "CN=FleetDM",
			expected: &HostCertificateNameDetails{
				Country:            "",
				Organization:       "",
				OrganizationalUnit: "",
				CommonName:         "FleetDM",
			},
		},
		{
			name:  "with plusses as separator",
			input: "DN=something+CN=FleetDM+OU=Org",
			expected: &HostCertificateNameDetails{
				Country:            "",
				Organization:       "",
				OrganizationalUnit: "Org",
				CommonName:         "FleetDM",
			},
		},
		{
			name:  "with plusses inside values and slash as separator",
			input: "DN=something/CN=FleetDM+valid/OU=Org",
			expected: &HostCertificateNameDetails{
				Country:            "",
				Organization:       "",
				OrganizationalUnit: "Org",
				CommonName:         "FleetDM+valid",
			},
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
