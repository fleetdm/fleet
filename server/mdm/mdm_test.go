package mdm

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeCMS(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		out    []byte
		outErr string
	}{
		{
			"valid, decodable message",
			"MIAGCSqGSIb3DQEHA6CAMIACAQAxggFQMIIBTAIBADA0MC8xCTAHBgNVBAYTADEQMA4GA1UEChMHc2NlcC1jYTEQMA4GA1UECxMHU0NFUCBDQQIBATANBgkqhkiG9w0BAQEFAASCAQBusyskXHF7kvWwwBs44zMvQfbtoBOMtESu7Wb2kFlSLOWFJjeb5wMUtmP6OUEYiNC5qP5Ig7m2KHRBtmdG/bUNH+UZQrn84PCIImwNeXl7o0NE5qxQda+R/n4gxCAc5XzjYPCfYVQsFiC2qVps5QPyNlXLoR7hnEm0nSgVeDYW6qg0vg5mZYRZZ01RE2T8HYEUWgjIRiVPbDkj/sGRASsFGuRlCiMx0fdkfH+fB4ir9VvUpW38GjFVkvnGoIflperEaKk6uOfQIzwFpV9kn+xVFxWSq6jNp99ASne40mHIvUH8D/7a7qpfCFWF9RXkY4A/vUU8cQtqUryvqReaWKHHMIAGCSqGSIb3DQEHATAUBggqhkiG9w0DBwQIjCknWIVENfWggAQYGVn0ydCHbO/umUeDFBbz620zBAHJ3yUcBAhglCmprZMW8gAAAAAAAAAAAAA=",
			[]byte("5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT"),
			"",
		},
		{
			"valid message encrypted using a different cert",
			"MIAGCSqGSIb3DQEHA6CAMIACAQAxggFtMIIBaQIBADBRMEgxHzAdBgNVBAMMFkZpbGVWYXVsdCBSZWNvdmVyeSBLZXkxJTAjBgNVBA0MHFJvYmVydG9zLU1hY0Jvb2stUHJvLTIubG9jYWwCBQCovxm3MA0GCSqGSIb3DQEBAQUABIIBAHiz8IGpXp+vqfTes7ejbvS11XpnaHCxDeaMYjmEJgZKtwdQhOJZy9clsypwqFv6h/Cva3/SuOEcwBoS2N/YY766jDP8nU4OcUaZWqEcMhRsSs1mil4T+rTnUfQEUKU9xW1j/iFq3xVWDTaBY+5cBgwUmdZb8XoWhXUVoF73OD0NpitnXxsxHokXv+UZzPoydlsCzhfAngl11hELAuFe6/mfq801E1hT+zvzDEDvfwSBMDC14OGDoFORVe/HCBS3NFGpVV+IrqpIpT1wbNx2dazmngduviErpXTgZG2vrCMQN1rN0OeLRtOMcjE6rer+ruuc5hfvTGMwWOgteqd2YQUwgAYJKoZIhvcNAQcBMBQGCCqGSIb3DQMHBAhwRO3eyigWMaCABBhy88Lm9qisQ9sOaf8u8GSzoWFdw2LkjRMECAKJG0H5K6iTAAAAAAAAAAAAAA==",
			nil,
			"pkcs7: no enveloped recipient for provided certificate",
		},
		{
			"invalid message",
			"invalid",
			nil,
			"illegal base64 data at input byte 4",
		},
		{
			"empty message",
			"",
			nil,
			"pkcs7: input data is empty",
		},
	}

	cert, err := tls.X509KeyPair(testSCEPCert, testSCEPKey)
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key, err := DecryptBase64CMS(c.in, parsed, cert.PrivateKey)
			require.EqualValues(t, c.out, key)
			if c.outErr != "" {
				require.EqualError(t, err, c.outErr)
			}
		})
	}
}

var (
	testSCEPCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDGzCCAgOgAwIBAgIBATANBgkqhkiG9w0BAQsFADAvMQkwBwYD
VQQGEwAxEDAOBgNVBAoTB3NjZXAtY2ExEDAOBgNVBAsTB1NDRVAg
Q0EwHhcNMjIxMjIyMTM0NDMzWhcNMzIxMjIyMTM0NDMzWjAvMQkw
BwYDVQQGEwAxEDAOBgNVBAoTB3NjZXAtY2ExEDAOBgNVBAsTB1ND
RVAgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDV
u9YVfl7gu0UgUkOJoES/XrN0WZdIjgvS2upKfvP4LSJOq1Mnp3bH
wWOA2NkHem/kjOVeotOk1aEYIzxbic6VlvNOz9huOhbJyoV4TO5v
tp/GFFcJ4IXh+f1Q4vm/NeH/XxEWn9S20B9OkSMOUievYsAu6iSi
oWaa74q1mnfpzM29p3dNM82mCKutYdkW0EusixU/CQxcVhdcxC+R
RyM4jzBFIipa7H20UtqdkZ03/9BoowJb/h/r4X7TN4tKg2vcwpZK
uJo7VcTBNPxhBowzg3JUmzjCnxPbuU/Ow5kPGOLJtbf4766ToNTM
/J63i3UPshKUBqAE8mIZO3qb7s25AgMBAAGjQjBAMA4GA1UdDwEB
/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTxPEY4
WvsLCt+HDQfnEPOKrHu0gTANBgkqhkiG9w0BAQsFAAOCAQEAGNf5
R60vRxIfvSOUyV3X7lUk+fVvi1CKC43DsP5OsQ6g5YVGcVXN40U4
2o7JUeb9K1jvqnzWB/3k+lSCkEb0a5KabjZE5Vpdt9xctmgrfNnQ
PBCfDdyb0Upjm61CJeB2SW9+ibT2L+OtL/nZjjlugL7ir9ramQBh
0IY6oB9Yc3TyZyPjnXwbi0jv5cildzIYaYPvPkPPTjezOUqUDgUH
JtdWRBQeJ/6WxAAm9il0KVXOsRPgAsdiDJTF6FdW4lsY8V/R6y0H
hTN1ZSyqklKAuvEZZznfmJsrNYRII2Fv2zOk0Uv/+E+EKTOHbgcC
PQAARDBzDlWvlMGWcbdrdypdeA==
-----END CERTIFICATE-----
`)

	testSCEPKey = []byte(testingKey(`-----BEGIN RSA TESTING KEY-----
MIIEowIBAAKCAQEA1bvWFX5e4LtFIFJDiaBEv16zdFmXSI4L0trqSn7z+C0iTqtT
J6d2x8FjgNjZB3pv5IzlXqLTpNWhGCM8W4nOlZbzTs/YbjoWycqFeEzub7afxhRX
CeCF4fn9UOL5vzXh/18RFp/UttAfTpEjDlInr2LALuokoqFmmu+KtZp36czNvad3
TTPNpgirrWHZFtBLrIsVPwkMXFYXXMQvkUcjOI8wRSIqWux9tFLanZGdN//QaKMC
W/4f6+F+0zeLSoNr3MKWSriaO1XEwTT8YQaMM4NyVJs4wp8T27lPzsOZDxjiybW3
+O+uk6DUzPyet4t1D7ISlAagBPJiGTt6m+7NuQIDAQABAoIBAE6LXL1BV3SW3Wxn
TtKAx0Lcdm5HjkTnjojKUldWGCoXzAfFBiYIcKov83UiO394Cy6eaJxCkix9JVpN
eJzbI8PtWTSZRRwc1MsLVclD3EvJfSW5y9KhZBILYIAdKVKPZqIGOa1qxyz3hsnE
pHFa16KoU5/qA9SQI7jEVuEuBusv4D/dRlEWvva7QOhnLrBPrSnTSZ5LxCFKRviS
XrEQ9AuRJeXCKx4WzXd4IZPpgldYHMJSSGMr0TeVcURbsfveI2IWvOLag0ofTHhx
tolBT2sKzInItLTwt/irZEp5lV08mMGxHuxoCdzhxjFQP8eGOZzPW65c6/D9hEXd
DzWnjdECgYEA9QtTQosOTtAyU1i4Fm76ltT6nywHy23KAMhBaoKgTMccNtjaOCg/
5FCCRD+qoo7TF4jdliP2NrMIbAIhr4jEfHSMKaD/rae1xqInseDCrGi9gzvm8UxG
84VG30Id8s70ZQWZjR/PFFDeNZjNhlk8COO0XoLaqJSZr+A30aSyeUsCgYEA30ok
3EvO1+/gjZv28J9vApdbiEwtO9xoteghElFzdtuEuzA+wL83w8xvKvdb4Rk5xigE
6mV69dBPj8zSyGp0lFTYLFvry5N4S8L6QPzt2nk+Lc3cDKSA5CkAkQ5Dmt5JwhxF
qIPDNZGXmoldIWJ0p/ZSu98/1yXBMQ9gCje/losCgYBwuk4KLbheT27nYsgFIfbL
zpyg/vty/UXRiE53tjISQALdxHLXJMUHvnW++d8Au12m1QLDIDYTQdddALoIa42g
h2k3eWZFuAJqp4xFS1WjROfx6Gu8k8+MFcLd0CfA3K4XjzTtdDWqbe1bkLjz1jdF
C6OdWutGZF4zR53GJtMn8wKBgCfA95cRGB5x4rTTk797YzQ+5lj51wPVVf8s+NZe
EgSTSKpbCJEgejkt6IzpxT3qU9LnxRhGQQIKuF+Nw+lSqrbN9D7RjsWL19sFN7Di
VyaSd3OINyk5EImOkz9AHuEvukoI5o3+B38+EJO+6QnMkaBlxo0UTjVrz12As0Se
cEnJAoGBAOUXjez9oUSzLzqG/WJFrIfHyjDA1vBS1j39XuhDuJGqMdNLlCE8Yr7h
d3gpZeuV3ZC33QAuwAXfRBNnKIDtDGpcrozM1NndcBVDs9GYvobaTiUaODGjsH44
oHwpyQbv9Qs+3bjPOQ7DkwekT+w1cptEKudBCC3WQKui1P0NNL0R
-----END RSA PRIVATE KEY-----
`))
)

// prevent static analysis tools from raising issues due to detection of private key
// in code.
func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

func TestGetRawProfilePlatform(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Darwin case sensitive",
			input:    []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"),
			expected: "darwin",
		},
		{
			name:     "Darwin case insensitive",
			input:    []byte("<?XML version=\"1.0\" encoding=\"UTF-8\"?>"),
			expected: "darwin",
		},
		{
			name:     "Windows case sensitive",
			input:    []byte("<Replace this=\"that\">"),
			expected: "windows",
		},
		{
			name:     "Windows case insensitive",
			input:    []byte("<REPLACE this=\"that\">"),
			expected: "windows",
		},
		{
			name:     "Windows case insensitive add ",
			input:    []byte("<ADD this=\"that\">"),
			expected: "windows",
		},
		{
			name:     "Windows case sensitive add",
			input:    []byte("<Add this=\"that\">"),
			expected: "windows",
		},
		{
			name:     "Whitespace before prefix",
			input:    []byte("   <?xml version=\"1.0\"?>"),
			expected: "darwin",
		},
		{
			name:     "Non-matching prefix",
			input:    []byte("<nonmatching>"),
			expected: "",
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "Only whitespaces",
			input:    []byte("   "),
			expected: "",
		},
		{
			name:     "Partial match",
			input:    []byte("<?x"),
			expected: "",
		},
		{
			name:     "DDM JSON",
			input:    []byte(`{"foo": "bar"}`),
			expected: "darwin",
		},
		{
			name:     "DDM JSON with whitespace",
			input:    []byte(`     {"foo": "bar"}`),
			expected: "darwin",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRawProfilePlatform(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestGuessProfileExtension(t *testing.T) {
	testCases := []struct {
		name     string
		profile  []byte
		expected string
	}{
		{
			name:     "XML with <?xml prefix",
			profile:  []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"),
			expected: "xml",
		},
		{
			name:     "XML with <replace prefix",
			profile:  []byte("<replace value=\"something\"/>"),
			expected: "xml",
		},
		{
			name:     "XML with <add prefix",
			profile:  []byte("<add key=\"somekey\" value=\"somevalue\"/>"),
			expected: "xml",
		},
		{
			name:     "JSON with { prefix",
			profile:  []byte("{ \"key\": \"value\" }"),
			expected: "json",
		},
		{
			name:     "Empty string",
			profile:  []byte(""),
			expected: "",
		},
		{
			name:     "Text with no recognizable prefix",
			profile:  []byte("This is just some text."),
			expected: "",
		},
		{
			name:     "XML with spaces before prefix",
			profile:  []byte("   <?xml version=\"1.0\" encoding=\"UTF-8\"?>"),
			expected: "xml",
		},
		{
			name:     "JSON with spaces before prefix",
			profile:  []byte("   { \"key\": \"value\" }"),
			expected: "json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GuessProfileExtension(tc.profile)
			require.Equal(t, tc.expected, result, "Expected result does not match actual result")
		})
	}
}
