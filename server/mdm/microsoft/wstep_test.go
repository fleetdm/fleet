package microsoft_mdm

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"crypto/x509"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/stretchr/testify/require"
)

type mockStore struct{}

func (m *mockStore) WSTEPStoreCertificate(ctx context.Context, name string, crt *x509.Certificate) error {
	return nil
}

func (m *mockStore) WSTEPNewSerial(ctx context.Context) (*big.Int, error) {
	return nil, nil
}

func (m *mockStore) WSTEPAssociateCertHash(ctx context.Context, deviceUUID string, hash string) error {
	return nil
}

var _ CertStore = (*mockStore)(nil)

func TestNewCertManager(t *testing.T) {
	var store CertStore

	wantCert, err := cryptoutil.DecodePEMCertificate(testCert)
	require.NoError(t, err)
	wantKey, err := server.DecodePrivateKeyPEM(testKey)
	require.NoError(t, err)
	wantIdentityFingerprint := CertFingerprintHexStr(wantCert)

	// Test that NewCertManager returns an error if the cert PEM is invalid.
	_, err = NewCertManager(store, []byte("invalid"), testKey)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to decode PEM certificate")

	// Test that NewCertManager returns an error if the key PEM is invalid.
	_, err = NewCertManager(store, testCert, []byte("invalid"))
	require.Error(t, err)
	require.ErrorContains(t, err, "decode private key: no PEM-encoded data found")

	// Test that NewCertManager returns an error if the cert PEM is not a certificate.
	_, err = NewCertManager(store, testKey, testKey)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to decode PEM certificate")

	// Test that NewCertManager returns an error if the key PEM is not a private key.
	_, err = NewCertManager(store, testCert, testCert)
	require.Error(t, err)
	require.ErrorContains(t, err, "decode private key: unexpected block type")

	// Test that NewCertManager returns a *WSTEPDepot if the cert and key PEMs are valid.
	cm, err := NewCertManager(store, testCert, testKey)
	require.NoError(t, err)
	require.NotNil(t, cm)
	require.Equal(t, wantIdentityFingerprint, cm.IdentityFingerprint())

	// Test that newManager sets the correct fields.
	m := cm.(*manager)
	require.NoError(t, err)
	require.Equal(t, *wantCert, *m.identityCert)
	require.NoError(t, err)
	require.Equal(t, *wantKey, *m.identityPrivateKey)
	require.Equal(t, wantIdentityFingerprint, m.identityFingerprint)
}

func TestSTSTokenSigningAndVerification(t *testing.T) {
	var store CertStore

	cm, err := NewCertManager(store, testCert, testKey)
	require.NoError(t, err)
	require.NotNil(t, cm)

	// Get a New STS Auth token
	upnEmail := "test@email.com"
	stsToken, err := cm.NewSTSAuthToken(upnEmail)
	require.NoError(t, err)
	require.NotEmpty(t, stsToken)

	// Verify the STS Auth token
	upnToken, err := cm.GetSTSAuthTokenUPNClaim(stsToken)
	require.NoError(t, err)
	require.NotEmpty(t, upnToken)
	require.Equal(t, upnEmail, upnToken)

	// New invalid STS Auth token
	_, err = cm.NewSTSAuthToken("")
	require.ErrorContains(t, err, "invalid upn field")
}

func TestCertFingerprintHexStr(t *testing.T) {
	cases := []struct {
		name string
		cert []byte
		err  error
	}{
		{
			name: "valid cert",
			cert: testCert,
			err:  nil,
		},
		{
			name: "invalid cert",
			cert: []byte("invalid"),
			err:  errors.New("failed to decode PEM certificate"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := cryptoutil.DecodePEMCertificate(tc.cert)
			if tc.err != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.err.Error())
				return
			}

			require.NoError(t, err)
			csum := sha1.Sum(cert.Raw) // nolint:gosec
			want := strings.ToUpper(hex.EncodeToString(csum[:]))
			fp := CertFingerprintHexStr(cert)
			require.Equal(t, want, fp)
		})
	}
}

var (
	testCert = []byte(`-----BEGIN CERTIFICATE-----
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

	testKey = []byte(testingKey(`-----BEGIN RSA TESTING KEY-----
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
