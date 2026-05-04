package service

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mock"
	mysql_errors "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/require"
)

// euaTestingKey replaces "TESTING KEY" with "PRIVATE KEY" to prevent secret
// scanners from flagging test keys embedded in source files.
func euaTestingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

// testWSTEPCert and testWSTEPKey are the same certs used in wstep_test.go.
var (
	testWSTEPCert = []byte(`-----BEGIN CERTIFICATE-----
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

	testWSTEPKey = []byte(euaTestingKey(`-----BEGIN RSA TESTING KEY-----
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
-----END RSA TESTING KEY-----
`))
)

// newTestServiceWithWSTEP returns a Service with a real wstepCertManager built
// from the inline test cert/key, backed by a mock datastore.
func newTestServiceWithWSTEP(t *testing.T, ds *mock.Store) *Service {
	t.Helper()
	certManager, err := microsoft_mdm.NewCertManager(nil, testWSTEPCert, testWSTEPKey)
	require.NoError(t, err)

	return &Service{
		ds:               ds,
		wstepCertManager: certManager,
		logger:           slog.New(slog.DiscardHandler),
	}
}

func TestProcessWindowsEUAToken(t *testing.T) {
	const (
		testUPN      = "user@example.com"
		testDeviceID = "device-abc-123"
		testHostUUID = "host-uuid-xyz"
		testAcctUUID = "acct-uuid-456"
	)

	// Helper to generate a valid token for test cases.
	makeToken := func(t *testing.T, svc *Service, upn, deviceID string) string {
		t.Helper()
		tok, err := svc.wstepCertManager.NewEUAToken(upn, deviceID)
		require.NoError(t, err)
		return tok
	}

	t.Run("valid token, new enrollment, account not yet in db", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)
		token := makeToken(t, svc, testUPN, testDeviceID)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			require.Equal(t, testDeviceID, mdmDeviceID)
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, HostUUID: ""}, nil
		}
		// First call returns not-found; second call (after insert) returns the account.
		getByEmailCalls := 0
		ds.GetMDMIdPAccountByEmailFunc = func(ctx context.Context, email string) (*fleet.MDMIdPAccount, error) {
			require.Equal(t, testUPN, email)
			getByEmailCalls++
			if getByEmailCalls == 1 {
				return nil, mysql_errors.NotFound("MDMIdPAccount")
			}
			return &fleet.MDMIdPAccount{UUID: testAcctUUID, Email: testUPN, Username: testUPN}, nil
		}
		ds.InsertMDMIdPAccountFunc = func(ctx context.Context, account *fleet.MDMIdPAccount) error {
			require.Equal(t, testUPN, account.Email)
			return nil
		}
		ds.AssociateHostMDMIdPAccountDBFunc = func(ctx context.Context, hostUUID, acctUUID string) error {
			require.Equal(t, testHostUUID, hostUUID)
			require.Equal(t, testAcctUUID, acctUUID)
			return nil
		}

		upn, deviceID, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, token)
		require.NoError(t, err)
		require.Equal(t, testUPN, upn)
		require.Equal(t, testDeviceID, deviceID)
		require.True(t, ds.AssociateHostMDMIdPAccountDBFuncInvoked)
	})

	t.Run("valid token, account already exists in db", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)
		token := makeToken(t, svc, testUPN, testDeviceID)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, HostUUID: ""}, nil
		}
		// Account exists — Insert should NOT be called.
		ds.GetMDMIdPAccountByEmailFunc = func(ctx context.Context, email string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{UUID: testAcctUUID, Email: testUPN, Username: "existing-username", Fullname: "Existing Name"}, nil
		}
		ds.AssociateHostMDMIdPAccountDBFunc = func(ctx context.Context, hostUUID, acctUUID string) error {
			return nil
		}

		_, _, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, token)
		require.NoError(t, err)
		require.False(t, ds.InsertMDMIdPAccountFuncInvoked, "should not insert when account already exists")
		require.True(t, ds.AssociateHostMDMIdPAccountDBFuncInvoked)
	})

	t.Run("valid token, enrollment already has host_uuid — still links idp account", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)
		token := makeToken(t, svc, testUPN, testDeviceID)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			// HostUUID already set — device was previously enrolled.
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, HostUUID: "existing-host-uuid"}, nil
		}
		// Account already exists — re-enrollment after host deletion may
		// have left the enrollment row populated but the mapping missing.
		ds.GetMDMIdPAccountByEmailFunc = func(ctx context.Context, email string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{UUID: testAcctUUID, Email: testUPN, Username: testUPN}, nil
		}
		ds.AssociateHostMDMIdPAccountDBFunc = func(ctx context.Context, hostUUID, acctUUID string) error {
			require.Equal(t, testHostUUID, hostUUID)
			require.Equal(t, testAcctUUID, acctUUID)
			return nil
		}

		upn, deviceID, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, token)
		require.NoError(t, err)
		require.Equal(t, testUPN, upn)
		require.Equal(t, testDeviceID, deviceID)
		require.True(t, ds.GetMDMIdPAccountByEmailFuncInvoked, "should still fetch idp account even when enrollment has host_uuid")
		require.True(t, ds.AssociateHostMDMIdPAccountDBFuncInvoked, "should still link idp account even when enrollment has host_uuid")
	})

	t.Run("invalid token falls back to END_USER_AUTH_REQUIRED", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)

		_, _, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, "this.is.not.a.valid.token")
		require.Error(t, err)
		var orbitErr *fleet.OrbitError
		require.ErrorAs(t, err, &orbitErr)
		require.Equal(t, "END_USER_AUTH_REQUIRED", orbitErr.Message)
		require.False(t, ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFuncInvoked)
	})

	t.Run("nil wstepCertManager falls back to END_USER_AUTH_REQUIRED without panic", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, _, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, "any.token.value")
		require.Error(t, err)
		var orbitErr *fleet.OrbitError
		require.ErrorAs(t, err, &orbitErr)
		require.Equal(t, "END_USER_AUTH_REQUIRED", orbitErr.Message)
		require.False(t, ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFuncInvoked)
	})

	t.Run("device not found falls back to END_USER_AUTH_REQUIRED", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)
		token := makeToken(t, svc, testUPN, testDeviceID)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, mysql_errors.NotFound("MDMWindowsEnrolledDevice")
		}

		_, _, err := svc.processWindowsEUAToken(context.Background(), testHostUUID, token)
		require.Error(t, err)
		var orbitErr *fleet.OrbitError
		require.ErrorAs(t, err, &orbitErr)
		require.Equal(t, "END_USER_AUTH_REQUIRED", orbitErr.Message)
	})
}

func TestGenerateWindowsEUAToken(t *testing.T) {
	const (
		testUPN      = "user@example.com"
		testDeviceID = "device-abc-123"
	)

	t.Run("returns token for device with valid UPN", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, MDMEnrollUserID: testUPN}, nil
		}

		token := svc.generateWindowsEUAToken(context.Background(), testDeviceID)
		require.NotEmpty(t, token)

		// Token should be valid and contain expected claims.
		claims, err := svc.wstepCertManager.GetEUATokenClaims(token)
		require.NoError(t, err)
		require.Equal(t, testUPN, claims.UPN)
		require.Equal(t, testDeviceID, claims.DeviceID)
	})

	t.Run("returns empty string when device has no UPN", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, MDMEnrollUserID: ""}, nil
		}

		require.Empty(t, svc.generateWindowsEUAToken(context.Background(), testDeviceID))
	})

	t.Run("returns empty string when device not found", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, mysql_errors.NotFound("MDMWindowsEnrolledDevice")
		}

		require.Empty(t, svc.generateWindowsEUAToken(context.Background(), testDeviceID))
	})

	t.Run("returns empty string when datastore returns error", func(t *testing.T) {
		ds := new(mock.Store)
		svc := newTestServiceWithWSTEP(t, ds)

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, sql.ErrConnDone
		}

		require.Empty(t, svc.generateWindowsEUAToken(context.Background(), testDeviceID))
	})

	t.Run("returns empty string when wstepCertManager is nil", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{MDMDeviceID: testDeviceID, MDMEnrollUserID: testUPN}, nil
		}

		require.Empty(t, svc.generateWindowsEUAToken(context.Background(), testDeviceID))
		require.False(t, ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFuncInvoked, "should not query db when cert manager is nil")
	})
}
