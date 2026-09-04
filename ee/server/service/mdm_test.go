package service

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	nanomdm_mdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	mocksvc "github.com/fleetdm/fleet/v4/server/mock/service"
	svcmock "github.com/fleetdm/fleet/v4/server/service/mock"

	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanolib/log/stdlogfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func setup(t *testing.T) (*mock.Store, *Service) {
	ds := new(mock.Store)

	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert:   {Value: []byte(testCert)},
			fleet.MDMAssetCAKey:    {Value: []byte(testKey)},
			fleet.MDMAssetAPNSKey:  {Value: []byte(testKey)},
			fleet.MDMAssetAPNSCert: {Value: []byte(testCert)},
		}, nil
	}

	svc := &Service{
		ds: ds,
	}
	return ds, svc
}

func TestMDMAppleReconcileFileVaultProfile(t *testing.T) {
	ctx := context.Background()

	getPayloadWithType := func(mc mobileconfig.Mobileconfig, payloadType string) map[string]interface{} {
		var payload struct {
			PayloadContent []map[string]interface{}
		}
		_, err := plist.Unmarshal(mc, &payload)
		require.NoError(t, err)

		for _, p := range payload.PayloadContent {
			if p["PayloadType"] == payloadType {
				return p
			}
		}
		return nil
	}

	// withMacOSSettings points the reconciler at the given macOS settings for
	// no-team and for any fleet, so each case only states the settings it means
	withMacOSSettings := func(ds *mock.Store, enforcement, escrow bool) {
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.MacOSSettings.EnableDiskEncryption = optjson.SetBool(enforcement)
			ac.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(escrow)
			return ac, nil
		}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			tm := &fleet.TeamMDM{}
			tm.MacOSSettings.EnableDiskEncryption = optjson.SetBool(enforcement)
			tm.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(escrow)
			return tm, nil
		}
	}

	t.Run("fails if SCEP is not configured", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds}
		withMacOSSettings(ds, true, true)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
			_ sqlx.QueryerContext,
		) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, nil
		}
		err := svc.MDMAppleReconcileFileVaultProfile(ctx, nil)
		require.Error(t, err)
	})

	t.Run("fails if the profile can't be saved in the db", func(t *testing.T) {
		ds, svc := setup(t)
		withMacOSSettings(ds, true, true)
		testErr := errors.New("test")
		ds.UpsertMDMAppleFleetConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) error {
			return testErr
		}
		err := svc.MDMAppleReconcileFileVaultProfile(ctx, nil)
		require.ErrorIs(t, err, testErr)
		require.True(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
	})

	t.Run("both settings on upserts the full profile", func(t *testing.T) {
		var teamID uint = 4
		ds, svc := setup(t)
		withMacOSSettings(ds, true, true)
		ds.UpsertMDMAppleFleetConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) error {
			require.Equal(t, &teamID, p.TeamID)
			require.Equal(t, p.Identifier, mobileconfig.FleetFileVaultPayloadIdentifier)
			require.Equal(t, p.Name, mdm.FleetFileVaultProfileName)
			require.Contains(t, string(p.Mobileconfig), `MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUxEzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhvcml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoMClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VHcJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/Ufa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JSkPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVjE26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSwVeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTfUNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSndAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0HMOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLXKLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6ClieTTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=`)

			testPayload := getPayloadWithType(p.Mobileconfig, "com.apple.MCX.FileVault2")
			require.NotNil(t, testPayload)
			require.Equal(t, true, testPayload["Defer"])
			require.EqualValues(t, 0, testPayload["DeferForceAtUserLoginMaxBypassAttempts"])
			require.Equal(t, false, testPayload["ShowRecoveryKey"])

			return nil
		}

		err := svc.MDMAppleReconcileFileVaultProfile(ctx, new(teamID))
		require.NoError(t, err)
		require.True(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
	})

	t.Run("enforcement only carries no escrow payloads", func(t *testing.T) {
		ds, svc := setup(t)
		withMacOSSettings(ds, true, false)
		ds.UpsertMDMAppleFleetConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) error {
			require.NotNil(t, getPayloadWithType(p.Mobileconfig, "com.apple.MCX.FileVault2"))
			require.NotNil(t, getPayloadWithType(p.Mobileconfig, "com.apple.MCX"))
			require.Nil(t, getPayloadWithType(p.Mobileconfig, "com.apple.security.FDERecoveryKeyEscrow"))
			require.Nil(t, getPayloadWithType(p.Mobileconfig, "com.apple.security.pkcs1"))
			// no escrow means the user needs to be shown the key
			require.Equal(t, true, getPayloadWithType(p.Mobileconfig, "com.apple.MCX.FileVault2")["ShowRecoveryKey"])
			return nil
		}
		require.NoError(t, svc.MDMAppleReconcileFileVaultProfile(ctx, new(uint(4))))
		require.True(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
	})

	t.Run("escrow only carries no enforcement payloads", func(t *testing.T) {
		ds, svc := setup(t)
		withMacOSSettings(ds, false, true)
		ds.UpsertMDMAppleFleetConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) error {
			require.Nil(t, getPayloadWithType(p.Mobileconfig, "com.apple.MCX.FileVault2"))
			require.Nil(t, getPayloadWithType(p.Mobileconfig, "com.apple.MCX"))
			require.NotNil(t, getPayloadWithType(p.Mobileconfig, "com.apple.security.FDERecoveryKeyEscrow"))
			require.NotNil(t, getPayloadWithType(p.Mobileconfig, "com.apple.security.pkcs1"))
			return nil
		}
		require.NoError(t, svc.MDMAppleReconcileFileVaultProfile(ctx, new(uint(4))))
		require.True(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
	})

	t.Run("enforcement only does not need the CA certificate", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds}
		withMacOSSettings(ds, true, false)
		// the escrow payload is what carries the certificate, so an unreadable
		// CA asset must not stop an enforcement-only profile from rendering
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
			_ sqlx.QueryerContext,
		) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, errors.New("CA asset unavailable")
		}
		ds.UpsertMDMAppleFleetConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) error {
			require.NotContains(t, string(p.Mobileconfig), "com.apple.security.pkcs1")
			return nil
		}
		require.NoError(t, svc.MDMAppleReconcileFileVaultProfile(ctx, nil))
		require.True(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
		require.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked, "the CA asset should not be read at all")
	})

	t.Run("both settings off removes the profile", func(t *testing.T) {
		var wantTeamID uint = 4
		ds, svc := setup(t)
		withMacOSSettings(ds, false, false)
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			require.NotNil(t, teamID)
			require.Equal(t, wantTeamID, *teamID)
			require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, profileIdentifier)
			return nil
		}
		require.NoError(t, svc.MDMAppleReconcileFileVaultProfile(ctx, new(wantTeamID)))
		require.True(t, ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFuncInvoked)
		require.False(t, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked)
	})

	t.Run("both settings off tolerates an already-absent profile", func(t *testing.T) {
		ds, svc := setup(t)
		withMacOSSettings(ds, false, false)
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			return notFoundErr{}
		}
		require.NoError(t, svc.MDMAppleReconcileFileVaultProfile(ctx, nil))
		require.True(t, ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFuncInvoked)
	})
}

type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

var (
	testCert = `-----BEGIN CERTIFICATE-----
MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUx
EzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhv
cml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoM
ClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VH
cJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/U
fa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0
bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj
6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JS
kPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG
9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVj
E26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4
QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSw
VeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTf
UNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSn
dAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0H
MOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLX
KLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6Clie
TTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py
1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk
8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=
-----END CERTIFICATE-----`

	testKey = testingKey(`-----BEGIN RSA TESTING KEY-----
MIIEogIBAAKCAQEApIdwfwSNK+nc2bggcAXEb0Nmbz7SHUHBTQ5fRUdwn4J/5/qH
Rdgxwyj0sUSmdQlFEyMqsziaQjvk0gi580E34au2qpM9OJ3zB84a79R9rkQRTWFv
aPsCIyWkzqLT5nKxdI2j5Vz9w1lPQ9d/I60mdibqn1ZkzzggPW3Z5rRsvALXZxMq
/mjJtUa0keHvpNHEBHcV5D35JjfS9QDFibqhbJiGjytF0q1RJ7XWlePo6HHF9Nme
xJafiWjS4GldqLrde2J5NWnPviG1I0LVu6cga/v/+bkXrSwfkysXklKQ/F8J+Cqy
SLzggINJE2W+lP54w+LKAAAnzNjzBhRdFgVhNQIDAQABAoIBAAtUbFHC3XnVq+iu
PkWYkBNdX9NvTwbGvWnyAGuD5OSHFwnBfck4fwzCaD9Ay/mpPsF3nXwj/LNs7m/s
O+ndZty6d2S9qOyaK98wuTgkuNbkRxC+Ee73wgjrkbLNEax/32p4Sn4D7lGid8vj
LhUl2k0ult+MEnsWkVnJk8TITeiQaT2AHhMr3HKdaI86hJJfam3wEBiLBglnnKqA
TInMqHoudnFOn/C8iVCFuHCE0oo1dMalbc4rlZuRBqezVhbSMWPLypMVXQb7eixM
ScJ3m8+DooGDSIe+EW/afhN2VnFbrhQC9/DlxGfwTwsUseWv7pgp53ufyyAzzydn
2plW/4ECgYEA1Va5RzSUDxr75JX003YZiBcYrG268vosiNYWRhE7frvn5EorZBRW
t4R70Y2gcXA10aPHzpbq40t6voWtpkfynU3fyRzbBmwfiWLEgckrYMwtcNz8nhG2
ETAg4LXO9CufbwuDa66h76TpkBzQVNc5TSbBUr/apLDWjKPMz6qW7VUCgYEAxW4K
Yqp3NgJkC5DhuD098jir9AH96hGhUryOi2CasCvmbjWCgWdolD7SRZJfxOXFOtHv
7Dkp9glA1Cg/nSmEHKslaTJfBIWK+5rqVD6k6kZE/+4QQWQtUxXXVgGINnGrnPvo
6MlRJxqGUtYJ0GRTFJP4Py0gwuzf5BMIwe+fpGECgYAOhLRfMCjTTlbOG5ZpvaPH
Kys2sNEEMBpPxaIGaq3N1iPV2WZSjT/JhW6XuDevAJ/pAGhcmtCpXz2fMaG7qzHL
mr0cBqaxLTKIOvx8iKA3Gi4NfDyE1Ve6m7fhEv5eh4l2GSZ8cYn7sRFkCVH0NCFm
KrkFVKEgjBhNwefySf2zcQKBgHDVPgw7nlv4q9LMX6RbI98eMnAG/2XZ45gUeWcA
tAeBX3WXEVoBjoxDBwuJ5z/xjXHbb8JSvT+G9E0MH6cjhgSYb44aoqFD7TV0yP2S
u8/Ej0SxewrURO8aKXJW99Edz9WtRuRbwgyWJTSMbRlzbOPy2UrJ8NJWbHK9yiCE
YXmhAoGAA3QUiCCl11c1C4VsF68Fa2i7qwnty3fvFidZpW3ds0tzZdIvkpRLp5+u
XAJ5+zStdEGdnu0iXALQlY7ektawXguT/zYKg3nfS9RMGW6CxZotn4bqfQwDuttf
b1xn1jGQd/o0xFf9ojpDNy6vNojidQGHh6E3h0GYvxbnQmVNq5U=
-----END RSA TESTING KEY-----`)
)

// prevent static analysis tools from raising issues due to detection of
// private key in code.
func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

func TestCountABMTokensAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ctx := context.Background()
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := Service{ds: ds, authz: authorizer}

	ds.GetABMTokenCountFunc = func(ctx context.Context) (int, error) {
		return 5, nil
	}

	t.Run("CountABMTokens", func(t *testing.T) {
		cases := []struct {
			desc              string
			user              *fleet.User
			shoudFailWithAuth bool
		}{
			{"no role", test.UserNoRoles, true},
			{"gitops can read", test.UserGitOps, false},
			{"maintainer can read", test.UserMaintainer, false},
			{"observer can read", test.UserObserver, false},
			{"observer+ can read", test.UserObserverPlus, false},
			{"admin can read", test.UserAdmin, false},
			{"tm1 gitops can read", test.UserTeamGitOpsTeam1, false},
			{"tm1 maintainer can read", test.UserTeamMaintainerTeam1, false},
			{"tm1 observer can read", test.UserTeamObserverTeam1, false},
			{"tm1 observer+ can read", test.UserTeamObserverPlusTeam1, false},
			{"tm1 admin can read", test.UserTeamAdminTeam1, false},
		}
		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				ctx = test.UserContext(ctx, c.user)
				count, err := svc.CountABMTokens(ctx)
				checkAuthErr(t, c.shoudFailWithAuth, err)
				if !c.shoudFailWithAuth {
					assert.EqualValues(t, 5, count)
				}
			})
		}
	})
}

func TestClearPasscode(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	// Set up the real commander with mocked storage and pusher.
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushProvider := &svcmock.APNSPushProvider{}
	pushProvider.PushFunc = func(_ context.Context, pushes []*nanomdm_mdm.Push) (map[string]*nanomdm_push.Response, error) {
		res := make(map[string]*nanomdm_push.Response, len(pushes))
		for _, p := range pushes {
			res[p.Token.String()] = &nanomdm_push.Response{Id: "ok"}
		}
		return res, nil
	}
	pushFactory := &svcmock.APNSPushProviderFactory{}
	pushFactory.NewPushProviderFunc = func(*tls.Certificate) (nanomdm_push.PushProvider, error) {
		return pushProvider, nil
	}
	pusher := nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushFactory, stdlogfmt.New())
	commander := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	svc := Service{ds: ds, authz: authorizer, mdmAppleCommander: commander, Service: &mocksvc.Service{
		NewActivityFunc: func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		},
	}}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMApplePermissions, error) {
		return &fleet.HostMDMApplePermissions{HostUUID: hostUUID, AccessRights: apple_mdm.MDMAccessRightAll}, nil
	}

	// Common mdmStorage mocks for enqueue + push.
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *nanomdm_mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, nil
	}
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, targets []string) (map[string]*nanomdm_mdm.Push, error) {
		pushes := make(map[string]*nanomdm_mdm.Push, len(targets))
		for _, uuid := range targets {
			pushes[uuid] = &nanomdm_mdm.Push{
				PushMagic: "magic" + uuid,
				Token:     []byte("token" + uuid),
				Topic:     "topic" + uuid,
			}
		}
		return pushes, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("../../../server/service/testdata/server.pem", "../../../server/service/testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}

	t.Run("authorization", func(t *testing.T) {
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}
		ds.GetNanoMDMEnrollmentDetailsFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoMDMEnrollmentDetails, error) {
			return &fleet.NanoMDMEnrollmentDetails{UnlockToken: new("fake-token")}, nil
		}

		team1 := new(uint(1))

		cases := []struct {
			desc       string
			user       *fleet.User
			hostTeamID *uint
			wantErr    error
		}{
			{"no role", test.UserNoRoles, nil, test.ErrForbidden},
			{"observer", test.UserObserver, nil, test.ErrForbidden},
			{"observer+", test.UserObserverPlus, nil, test.ErrForbidden},
			{"technician", test.UserTechnician, nil, nil},
			{"gitops", test.UserGitOps, nil, test.ErrForbidden},
			{"maintainer", test.UserMaintainer, nil, nil},
			{"admin", test.UserAdmin, nil, nil},
			{"team 1 technician", test.UserTeamTechnicianTeam1, team1, nil},
			{"team 1 admin", test.UserTeamAdminTeam1, team1, nil},
			{"team 1 observer", test.UserTeamObserverTeam1, team1, test.ErrForbidden},
			{"team 2 technician", test.UserTeamTechnicianTeam2, team1, test.ErrNotFound},
		}
		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
					return &fleet.Host{ID: hostID, Platform: "ipados", TeamID: c.hostTeamID}, nil
				}

				ctx := test.UserContext(t.Context(), c.user)
				_, err := svc.ClearPasscode(ctx, 1)
				test.RequireErrKind(t, c.wantErr, err)
			})
		}
	})

	t.Run("access rights disallow clear passcode", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "host-uuid-rights", Platform: "ipados"}, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMApplePermissions, error) {
			return &fleet.HostMDMApplePermissions{
				HostUUID:     hostUUID,
				AccessRights: apple_mdm.MDMAccessRightAll &^ apple_mdm.MDMAccessRightDeviceLock,
			}, nil
		}
		t.Cleanup(func() {
			ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMApplePermissions, error) {
				return &fleet.HostMDMApplePermissions{HostUUID: hostUUID, AccessRights: apple_mdm.MDMAccessRightAll}, nil
			}
		})

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		require.Contains(t, badReq.Message, fleet.CantClearPasscodeAccessRightsMessage)
	})

	t.Run("authorization non-Apple-mobile platforms", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "android"}, nil
		}

		// Technicians can only clear passcodes on iOS/iPadOS hosts; Android still
		// requires full MDM command write.
		ctx := test.UserContext(t.Context(), test.UserTechnician)
		_, err := svc.ClearPasscode(ctx, 1)
		checkAuthErr(t, true, err)

		// Same for macOS: it is not an Apple mobile platform, so technicians are
		// denied at the authorization gate rather than by platform validation.
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "darwin"}, nil
		}
		_, err = svc.ClearPasscode(ctx, 1)
		checkAuthErr(t, true, err)

		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "android"}, nil
		}

		// Admin passes authorization and fails on Android MDM not being configured,
		// proving the gate itself lets admins through.
		ctx = test.UserContext(t.Context(), test.UserAdmin)
		_, err = svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var forbiddenError *authz.Forbidden
		require.NotErrorAs(t, err, &forbiddenError)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		require.Contains(t, badReq.Message, fleet.AndroidMDMNotConfiguredMessage)
	})

	t.Run("happy path ipados", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "host-uuid-1", Platform: "ipados"}, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.NoError(t, err)
		require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
		mdmStorage.EnqueueCommandFuncInvoked = false
	})

	t.Run("happy path ios", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "host-uuid-2", Platform: "ios"}, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.NoError(t, err)
		require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
		mdmStorage.EnqueueCommandFuncInvoked = false
	})

	t.Run("non-apple platform", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "windows"}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Contains(t, badReq.Message, "only supported on Apple mobile platforms")
	})

	t.Run("macOS not supported", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "darwin"}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Contains(t, badReq.Message, "ClearPasscode command is only available for iOS and iPadOS. Unable to issue ClearPasscode command.")
	})

	t.Run("MDM not enabled", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "ipados"}, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: false}}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Contains(t, badReq.Message, "Apple MDM must be turned on to use Clear passcode.")

		// Restore for subsequent tests.
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
	})

	t.Run("personal enrollment", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "ipados"}, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{IsPersonalEnrollment: true}, nil
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Contains(t, badReq.Message, "Unlock token is not available")
	})

	t.Run("enqueue command error", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "host-uuid-3", Platform: "ipados"}, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}
		mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *nanomdm_mdm.CommandWithSubtype) (map[string]error, error) {
			return nil, errors.New("enqueue failed")
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "enqueue failed")

		// Restore for subsequent tests.
		mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *nanomdm_mdm.CommandWithSubtype) (map[string]error, error) {
			return nil, nil
		}
	})

	t.Run("host not found", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return nil, &notFoundError{}
		}

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		_, err := svc.ClearPasscode(ctx, 999)
		require.Error(t, err)
	})
}

func TestCancelHostMDMCommand(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	var lastActivity fleet.ActivityDetails
	svc := Service{ds: ds, authz: authorizer, Service: &mocksvc.Service{
		NewActivityFunc: func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			lastActivity = activity
			return nil
		},
	}}

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		return &fleet.Host{ID: hostID, UUID: "host-uuid-1", Platform: "darwin", Hostname: "mac-1"}, nil
	}
	ds.CancelHostMDMCommandFunc = func(ctx context.Context, host *fleet.Host, commandUUID string) (string, error) {
		return "DeviceLock", nil
	}

	t.Run("authorization", func(t *testing.T) {
		// The selective-list gate admits gitops (which can send raw MDM
		// commands via POST /commands/run); mdm_command write then decides.
		cases := []struct {
			desc              string
			user              *fleet.User
			shoudFailWithAuth bool
		}{
			{"no role", test.UserNoRoles, true},
			{"observer", test.UserObserver, true},
			{"observer+", test.UserObserverPlus, true},
			{"technician", test.UserTechnician, true},
			{"gitops", test.UserGitOps, false},
			{"maintainer", test.UserMaintainer, false},
			{"admin", test.UserAdmin, false},
		}
		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				ctx := test.UserContext(t.Context(), c.user)
				err := svc.CancelHostMDMCommand(ctx, 1, "cmd-uuid")
				checkAuthErr(t, c.shoudFailWithAuth, err)
			})
		}
	})

	t.Run("happy path", func(t *testing.T) {
		ds.CancelHostMDMCommandFuncInvoked = false
		lastActivity = nil

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		require.NoError(t, svc.CancelHostMDMCommand(ctx, 1, "cmd-uuid"))
		require.True(t, ds.CancelHostMDMCommandFuncInvoked)

		act, ok := lastActivity.(fleet.ActivityTypeCanceledMDMCommand)
		require.True(t, ok)
		assert.Equal(t, uint(1), act.HostID)
		assert.Equal(t, "mac-1", act.HostDisplayName)
		assert.Equal(t, "DeviceLock", act.CommandType)
	})

	t.Run("non-apple platform", func(t *testing.T) {
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, Platform: "windows"}, nil
		}
		ds.CancelHostMDMCommandFuncInvoked = false

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.CancelHostMDMCommand(ctx, 1, "cmd-uuid")
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Contains(t, badReq.Message, "Only Apple MDM commands can be canceled")
		require.False(t, ds.CancelHostMDMCommandFuncInvoked)

		// Restore for subsequent tests.
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "host-uuid-1", Platform: "darwin", Hostname: "mac-1"}, nil
		}
	})

	t.Run("datastore error means no activity", func(t *testing.T) {
		ds.CancelHostMDMCommandFunc = func(ctx context.Context, host *fleet.Host, commandUUID string) (string, error) {
			return "", &notFoundError{}
		}
		lastActivity = nil

		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.CancelHostMDMCommand(ctx, 1, "cmd-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.Nil(t, lastActivity)
	})
}

func TestUpdateABMTokenTeams(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(t.Context(), test.UserAdmin)

	// Set up the real commander with mocked storage and pusher.
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushProvider := &svcmock.APNSPushProvider{}
	pushProvider.PushFunc = func(_ context.Context, pushes []*nanomdm_mdm.Push) (map[string]*nanomdm_push.Response, error) {
		res := make(map[string]*nanomdm_push.Response, len(pushes))
		for _, p := range pushes {
			res[p.Token.String()] = &nanomdm_push.Response{Id: "ok"}
		}
		return res, nil
	}
	pushFactory := &svcmock.APNSPushProviderFactory{}
	pushFactory.NewPushProviderFunc = func(*tls.Certificate) (nanomdm_push.PushProvider, error) {
		return pushProvider, nil
	}
	pusher := nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushFactory, stdlogfmt.New())
	commander := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	svc := Service{ds: ds, authz: authorizer, mdmAppleCommander: commander, Service: &mocksvc.Service{
		NewActivityFunc: func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		},
	}}

	orgName := "Fake Organization"
	tokenID := uint(1)
	abmToken := &fleet.ABMToken{ID: tokenID, OrganizationName: orgName}
	ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
		return abmToken, nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true, AppleBusinessManager: optjson.SetSlice([]fleet.MDMAppleABMAssignmentInfo{
		{OrganizationName: orgName},
	})}}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appCfg, nil
	}

	var updatedAppCfg *fleet.AppConfig
	ds.SaveAppConfigFunc = func(ctx context.Context, cfg *fleet.AppConfig) error {
		updatedAppCfg = cfg
		return nil
	}

	validTeamID := new(uint(2))
	validTeamName := "Valid Team"
	invalidTeamID := new(uint(3))
	teamLiteCalls := 0
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		teamLiteCalls++
		if tid == *validTeamID {
			return &fleet.TeamLite{ID: *validTeamID, Name: validTeamName}, nil
		}
		return nil, &notFoundError{}
	}

	t.Run("team ids is validated and updated", func(t *testing.T) {
		teamLiteCalls = 0
		ds.SaveAppConfigFuncInvoked = false
		token, err := svc.UpdateABMTokenTeams(ctx, tokenID, validTeamID, validTeamID, validTeamID, validTeamID)
		require.NoError(t, err)

		assert.Equal(t, validTeamID, token.BYODDefaultTeamID)
		assert.Equal(t, validTeamID, token.MacOSDefaultTeamID)
		assert.Equal(t, validTeamID, token.IOSDefaultTeamID)
		assert.Equal(t, validTeamID, token.IPadOSDefaultTeamID)
		assert.Equal(t, 4, teamLiteCalls)
		require.True(t, ds.SaveAppConfigFuncInvoked)
		var appCfgToken fleet.MDMAppleABMAssignmentInfo
		for _, tok := range updatedAppCfg.MDM.AppleBusinessManager.Value {
			if tok.OrganizationName == orgName {
				appCfgToken = tok
				break
			}
		}
		assert.Equal(t, validTeamName, appCfgToken.BYODTeam)
		assert.Equal(t, validTeamName, appCfgToken.MacOSTeam)
		assert.Equal(t, validTeamName, appCfgToken.IOSTeam)
		assert.Equal(t, validTeamName, appCfgToken.IpadOSTeam)
	})

	t.Run("invalid team id returns error", func(t *testing.T) {
		teamLiteCalls = 0
		_, err := svc.UpdateABMTokenTeams(ctx, tokenID, validTeamID, validTeamID, validTeamID, invalidTeamID)
		require.Error(t, err)
	})

	t.Run("does not validate nil team ids", func(t *testing.T) {
		teamLiteCalls = 0
		ds.SaveAppConfigFuncInvoked = false
		appCfg.MDM.AppleBusinessManager = optjson.SetSlice([]fleet.MDMAppleABMAssignmentInfo{
			{OrganizationName: orgName, MacOSTeam: validTeamName, IOSTeam: validTeamName, IpadOSTeam: validTeamName, BYODTeam: validTeamName},
		})
		abmToken.MacOSDefaultTeamID = validTeamID
		abmToken.IOSDefaultTeamID = validTeamID
		abmToken.IPadOSDefaultTeamID = validTeamID
		abmToken.BYODDefaultTeamID = validTeamID
		abmToken.MacOSTeam.Name = validTeamName
		abmToken.MacOSTeam.ID = *validTeamID
		abmToken.IOSTeam.Name = validTeamName
		abmToken.IOSTeam.ID = *validTeamID
		abmToken.IPadOSTeam.Name = validTeamName
		abmToken.IPadOSTeam.ID = *validTeamID
		abmToken.BYODTeam.Name = validTeamName
		abmToken.BYODTeam.ID = *validTeamID
		token, err := svc.UpdateABMTokenTeams(ctx, tokenID, nil, nil, nil, nil)
		require.NoError(t, err)

		assert.Nil(t, token.BYODDefaultTeamID)
		assert.Nil(t, token.MacOSDefaultTeamID)
		assert.Nil(t, token.IOSDefaultTeamID)
		assert.Nil(t, token.IPadOSDefaultTeamID)
		assert.Equal(t, 0, teamLiteCalls) // no calls to TeamLite since all team ids are nil
		require.True(t, ds.SaveAppConfigFuncInvoked)
		var appCfgToken fleet.MDMAppleABMAssignmentInfo
		for _, tok := range updatedAppCfg.MDM.AppleBusinessManager.Value {
			if tok.OrganizationName == orgName {
				appCfgToken = tok
				break
			}
		}
		// Validate we clear out the "No team"
		assert.Empty(t, appCfgToken.BYODTeam)
		assert.Empty(t, appCfgToken.MacOSTeam)
		assert.Empty(t, appCfgToken.IOSTeam)
		assert.Empty(t, appCfgToken.IpadOSTeam)
	})

	t.Run("updates app config with new entry if not present", func(t *testing.T) {
		appCfg.MDM.AppleBusinessManager = optjson.SetSlice([]fleet.MDMAppleABMAssignmentInfo{})

		token, err := svc.UpdateABMTokenTeams(ctx, tokenID, validTeamID, validTeamID, validTeamID, validTeamID)
		require.NoError(t, err)

		assert.Equal(t, validTeamID, token.BYODDefaultTeamID)
		assert.Equal(t, validTeamID, token.MacOSDefaultTeamID)
		assert.Equal(t, validTeamID, token.IOSDefaultTeamID)
		assert.Equal(t, validTeamID, token.IPadOSDefaultTeamID)
		require.True(t, ds.SaveAppConfigFuncInvoked)
		var appCfgToken fleet.MDMAppleABMAssignmentInfo
		for _, tok := range updatedAppCfg.MDM.AppleBusinessManager.Value {
			if tok.OrganizationName == orgName {
				appCfgToken = tok
				break
			}
		}
		assert.Equal(t, validTeamName, appCfgToken.BYODTeam)
		assert.Equal(t, validTeamName, appCfgToken.MacOSTeam)
		assert.Equal(t, validTeamName, appCfgToken.IOSTeam)
		assert.Equal(t, validTeamName, appCfgToken.IpadOSTeam)
	})
}
func TestMDMAppleEditedAppleOSUpdatesDeclaration(t *testing.T) {
	ctx := context.Background()
	teamID := uint(1)

	// captured records what the datastore was handed, so the tests assert on the
	// generated declaration rather than on a real write.
	type captured struct {
		decl    *fleet.MDMAppleDeclaration
		vars    []fleet.FleetVarName
		deleted string
		labels  []string
	}

	newSvc := func() (*Service, *captured) {
		got := &captured{}
		ds := new(mock.Store)
		ds.LabelIDsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]uint, error) {
			got.labels = names
			ids := make(map[string]uint, len(names))
			for i, name := range names {
				ids[name] = uint(i + 1) //nolint:gosec
			}
			return ids, nil
		}
		ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, decl *fleet.MDMAppleDeclaration,
			usesFleetVars []fleet.FleetVarName, activationAction fleet.MDMAppleActivationAction,
		) (*fleet.MDMAppleDeclaration, error) {
			got.decl = decl
			got.vars = usesFleetVars
			decl.DeclarationUUID = "decl-uuid"
			return decl, nil
		}
		ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, declTeamID *uint, name string) error {
			got.deleted = name
			return nil
		}
		return &Service{ds: ds}, got
	}

	// Each platform gets its own declaration name and built-in label; a mix-up
	// would send the OS update declaration to the wrong devices.
	platforms := []struct {
		name      string
		device    fleet.AppleDevice
		declName  string
		labelName string
	}{
		{"macos", fleet.MacOS, mdm.FleetMacOSUpdatesProfileName, fleet.BuiltinLabelMacOS14Plus},
		{"ios", fleet.IOS, mdm.FleetIOSUpdatesProfileName, fleet.BuiltinLabelIOS},
		{"ipados", fleet.IPadOS, mdm.FleetIPadOSUpdatesProfileName, fleet.BuiltinLabelIPadOS},
	}

	t.Run("latest emits Fleet variable placeholders", func(t *testing.T) {
		for _, p := range platforms {
			t.Run(p.name, func(t *testing.T) {
				svc, got := newSvc()

				err := svc.mdmAppleEditedAppleOSUpdates(ctx, &teamID, p.device, fleet.AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
					DeadlineDays:   optjson.SetInt(14),
				})
				require.NoError(t, err)
				require.NotNil(t, got.decl)
				require.Empty(t, got.deleted)

				// The literal placeholder text matters: it is what gets substituted
				// per host at declaration fetch time.
				require.Contains(t, string(got.decl.RawJSON), `"TargetOSVersion": "$FLEET_VAR_HOST_TARGET_OS_VERSION"`)
				require.Contains(t, string(got.decl.RawJSON), `"TargetLocalDateTime": "${FLEET_VAR_HOST_TARGET_OS_DEADLINE}T12:00:00"`)
				// Without these the declaration is stored but never expanded.
				require.ElementsMatch(t, []fleet.FleetVarName{
					fleet.FleetVarHostTargetOSVersion,
					fleet.FleetVarHostTargetOSDeadline,
				}, got.vars)

				require.Equal(t, p.declName, got.decl.Name)
				require.Equal(t, []string{p.labelName}, got.labels)
			})
		}
	})

	t.Run("specific version emits literal values and no variables", func(t *testing.T) {
		for _, p := range platforms {
			t.Run(p.name, func(t *testing.T) {
				svc, got := newSvc()

				err := svc.mdmAppleEditedAppleOSUpdates(ctx, &teamID, p.device, fleet.AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("15.7.8"),
					Deadline:       optjson.SetString("2026-09-01"),
				})
				require.NoError(t, err)
				require.NotNil(t, got.decl)
				require.Contains(t, string(got.decl.RawJSON), `"TargetOSVersion": "15.7.8"`)
				require.Contains(t, string(got.decl.RawJSON), `"TargetLocalDateTime": "2026-09-01T12:00:00"`)
				require.NotContains(t, string(got.decl.RawJSON), "FLEET_VAR_")
				require.Empty(t, got.vars)

				require.Equal(t, p.declName, got.decl.Name)
				require.Equal(t, []string{p.labelName}, got.labels)
			})
		}
	})

	t.Run("disabled deletes the declaration", func(t *testing.T) {
		for _, p := range platforms {
			t.Run(p.name, func(t *testing.T) {
				svc, got := newSvc()

				err := svc.mdmAppleEditedAppleOSUpdates(ctx, &teamID, p.device, fleet.AppleOSUpdateSettings{})
				require.NoError(t, err)
				require.Nil(t, got.decl, "no declaration should be written when OS updates are off")
				require.Equal(t, p.declName, got.deleted)
			})
		}
	})
}
