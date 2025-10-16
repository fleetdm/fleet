package fleet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestExtractHostCertificateFromMDMAppleCertificateList(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	cases := []struct {
		name     string
		input    *x509.Certificate
		expected *HostCertificateRecord
		err      string
	}{
		{
			name: "multiple organizational unit values",
			input: &x509.Certificate{
				Subject: pkix.Name{
					Country:      []string{"US"},
					Organization: []string{"Fleet Device Management Inc."},
					OrganizationalUnit: []string{
						"Engineering",
						"Fleet Device Management Inc.",
						"fleet-a3ffb5cfa-3c69-433f-88af-d982ef9c3f67",
					},
					CommonName: "Test Multiple OU Values",
				},
			},
			expected: &HostCertificateRecord{
				SubjectCommonName:         "Test Multiple OU Values",
				SubjectCountry:            "US",
				SubjectOrganization:       "Fleet Device Management Inc.",
				SubjectOrganizationalUnit: "Engineering+OU=Fleet Device Management Inc.+OU=fleet-a3ffb5cfa-3c69-433f-88af-d982ef9c3f67",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			certDER, err := x509.CreateCertificate(rand.Reader, tc.input, tc.input, &privateKey.PublicKey, privateKey)
			require.NoError(t, err)
			li := []MDMAppleCertificateListItem{
				{
					Data: certDER,
				},
			}

			parsed, err := li[0].Parse(1)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, parsed)
				assert.Equal(t, tc.expected.SubjectCommonName, parsed.SubjectCommonName)
				assert.Equal(t, tc.expected.SubjectCountry, parsed.SubjectCountry)
				assert.Equal(t, tc.expected.SubjectOrganization, parsed.SubjectOrganization)
				assert.Equal(t, tc.expected.SubjectOrganizationalUnit, parsed.SubjectOrganizationalUnit)
			}
		})
	}
}
