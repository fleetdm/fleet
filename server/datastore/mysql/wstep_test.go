package mysql

import (
	"context"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestWSTEPStore(t *testing.T) {
	ds := CreateMySQLDS(t)

	wantCert, err := cryptoutil.DecodePEMCertificate(testCert)
	require.NoError(t, err)
	require.NoError(t, err)

	// serial number should start at 2 because 1 is reserved for the CA cert
	sn, err := ds.WSTEPNewSerial(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sn)
	require.Equal(t, int64(2), sn.Int64())

	// serial should increment
	sn, err = ds.WSTEPNewSerial(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sn)
	require.Equal(t, int64(3), sn.Int64())

	testCert := *wantCert
	testCert.SerialNumber = sn

	// store without setting a common name in the cert
	err = ds.WSTEPStoreCertificate(context.Background(), "test", &testCert)
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var dest []struct {
			Name    string `db:"name"`
			CertPEM []byte `db:"certificate_pem"`
		}
		err = sqlx.SelectContext(context.Background(), q, &dest, "SELECT name, certificate_pem FROM wstep_certificates where serial = 3")
		if err != nil {
			return err
		}
		require.Len(t, dest, 1)
		wantPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: testCert.Raw,
		})
		require.Equal(t, wantPEM, dest[0].CertPEM)

		// name wasn't set in the test cert, so it should default to sha256 of the cert
		require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(testCert.Raw)), dest[0].Name)

		return nil
	})

	// store with a common name in the cert
	testCert.Subject.CommonName = "test"
	err = ds.WSTEPStoreCertificate(context.Background(), "test", &testCert)
	require.Error(t, err) // duplicate serial number

	// get a new serial number
	sn, err = ds.WSTEPNewSerial(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sn)
	require.Equal(t, int64(4), sn.Int64())

	testCert.SerialNumber = sn
	err = ds.WSTEPStoreCertificate(context.Background(), "test", &testCert)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var dest []struct {
			Name    string `db:"name"`
			CertPEM []byte `db:"certificate_pem"`
		}
		err = sqlx.SelectContext(context.Background(), q, &dest, "SELECT name, certificate_pem FROM wstep_certificates where serial = 4")
		if err != nil {
			return err
		}
		require.Len(t, dest, 1)
		wantPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: testCert.Raw,
		})
		require.Equal(t, wantPEM, dest[0].CertPEM)

		// name wasn't set in the test cert, so it should default to sha256 of the cert
		require.Equal(t, "test", dest[0].Name)

		return nil
	})

	// TODO: test WSTEPAssociateCertHash when the intended usage is clear
}

var testCert = []byte(`-----BEGIN CERTIFICATE-----
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

//	var testKey = []byte(testingKey(`-----BEGIN RSA TESTING KEY-----
// MIIEowIBAAKCAQEA1bvWFX5e4LtFIFJDiaBEv16zdFmXSI4L0trqSn7z+C0iTqtT
// J6d2x8FjgNjZB3pv5IzlXqLTpNWhGCM8W4nOlZbzTs/YbjoWycqFeEzub7afxhRX
// CeCF4fn9UOL5vzXh/18RFp/UttAfTpEjDlInr2LALuokoqFmmu+KtZp36czNvad3
// TTPNpgirrWHZFtBLrIsVPwkMXFYXXMQvkUcjOI8wRSIqWux9tFLanZGdN//QaKMC
// W/4f6+F+0zeLSoNr3MKWSriaO1XEwTT8YQaMM4NyVJs4wp8T27lPzsOZDxjiybW3
// +O+uk6DUzPyet4t1D7ISlAagBPJiGTt6m+7NuQIDAQABAoIBAE6LXL1BV3SW3Wxn
// TtKAx0Lcdm5HjkTnjojKUldWGCoXzAfFBiYIcKov83UiO394Cy6eaJxCkix9JVpN
// eJzbI8PtWTSZRRwc1MsLVclD3EvJfSW5y9KhZBILYIAdKVKPZqIGOa1qxyz3hsnE
// pHFa16KoU5/qA9SQI7jEVuEuBusv4D/dRlEWvva7QOhnLrBPrSnTSZ5LxCFKRviS
// XrEQ9AuRJeXCKx4WzXd4IZPpgldYHMJSSGMr0TeVcURbsfveI2IWvOLag0ofTHhx
// tolBT2sKzInItLTwt/irZEp5lV08mMGxHuxoCdzhxjFQP8eGOZzPW65c6/D9hEXd
// DzWnjdECgYEA9QtTQosOTtAyU1i4Fm76ltT6nywHy23KAMhBaoKgTMccNtjaOCg/
// 5FCCRD+qoo7TF4jdliP2NrMIbAIhr4jEfHSMKaD/rae1xqInseDCrGi9gzvm8UxG
// 84VG30Id8s70ZQWZjR/PFFDeNZjNhlk8COO0XoLaqJSZr+A30aSyeUsCgYEA30ok
// 3EvO1+/gjZv28J9vApdbiEwtO9xoteghElFzdtuEuzA+wL83w8xvKvdb4Rk5xigE
// 6mV69dBPj8zSyGp0lFTYLFvry5N4S8L6QPzt2nk+Lc3cDKSA5CkAkQ5Dmt5JwhxF
// qIPDNZGXmoldIWJ0p/ZSu98/1yXBMQ9gCje/losCgYBwuk4KLbheT27nYsgFIfbL
// zpyg/vty/UXRiE53tjISQALdxHLXJMUHvnW++d8Au12m1QLDIDYTQdddALoIa42g
// h2k3eWZFuAJqp4xFS1WjROfx6Gu8k8+MFcLd0CfA3K4XjzTtdDWqbe1bkLjz1jdF
// C6OdWutGZF4zR53GJtMn8wKBgCfA95cRGB5x4rTTk797YzQ+5lj51wPVVf8s+NZe
// EgSTSKpbCJEgejkt6IzpxT3qU9LnxRhGQQIKuF+Nw+lSqrbN9D7RjsWL19sFN7Di
// VyaSd3OINyk5EImOkz9AHuEvukoI5o3+B38+EJO+6QnMkaBlxo0UTjVrz12As0Se
// cEnJAoGBAOUXjez9oUSzLzqG/WJFrIfHyjDA1vBS1j39XuhDuJGqMdNLlCE8Yr7h
// d3gpZeuV3ZC33QAuwAXfRBNnKIDtDGpcrozM1NndcBVDs9GYvobaTiUaODGjsH44
// oHwpyQbv9Qs+3bjPOQ7DkwekT+w1cptEKudBCC3WQKui1P0NNL0R
// -----END RSA PRIVATE KEY-----
// `)

// // prevent static analysis tools from raising issues due to detection of private key
// // in code.
// func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }
