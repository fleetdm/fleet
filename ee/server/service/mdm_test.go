package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

// mockMDMAppleCommander implements fleet.MDMAppleCommandIssuer for testing.
type mockMDMAppleCommander struct {
	clearPasscodeFunc func(ctx context.Context, host *fleet.Host, commandUUID string, unlockToken []byte) error
}

func (m *mockMDMAppleCommander) InstallProfile(_ context.Context, _ []string, _ mobileconfig.Mobileconfig, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) RemoveProfile(_ context.Context, _ []string, _ string, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) DeviceLock(_ context.Context, _ *fleet.Host, _ string) (string, error) {
	return "", nil
}
func (m *mockMDMAppleCommander) EnableLostMode(_ context.Context, _ *fleet.Host, _ string, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) DisableLostMode(_ context.Context, _ *fleet.Host, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) EraseDevice(_ context.Context, _ *fleet.Host, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) InstallEnterpriseApplication(_ context.Context, _ []string, _ string, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) DeviceConfigured(_ context.Context, _, _ string) error { return nil }
func (m *mockMDMAppleCommander) SetRecoveryLock(_ context.Context, _ []string, _ string) error {
	return nil
}
func (m *mockMDMAppleCommander) ClearPasscode(ctx context.Context, host *fleet.Host, commandUUID string, unlockToken []byte) error {
	if m.clearPasscodeFunc != nil {
		return m.clearPasscodeFunc(ctx, host, commandUUID, unlockToken)
	}
	return nil
}

// minimalMockFleetService provides the fleet.Service methods needed by ClearHostPasscode tests.
type minimalMockFleetService struct {
	fleet.Service
	newActivityCalled bool
}

func (s *minimalMockFleetService) VerifyMDMAppleConfigured(_ context.Context) error { return nil }
func (s *minimalMockFleetService) NewActivity(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
	s.newActivityCalled = true
	return nil
}

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

func TestMDMAppleEnableFileVaultAndEscrow(t *testing.T) {
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

	t.Run("fails if SCEP is not configured", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds}
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
			_ sqlx.QueryerContext,
		) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, nil
		}
		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, nil)
		require.Error(t, err)
	})

	t.Run("fails if the profile can't be saved in the db", func(t *testing.T) {
		ds, svc := setup(t)
		testErr := errors.New("test")
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile, vars []fleet.FleetVarName) (*fleet.MDMAppleConfigProfile, error) {
			return nil, testErr
		}
		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, nil)
		require.ErrorIs(t, err, testErr)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
	})

	t.Run("happy path", func(t *testing.T) {
		var teamID uint = 4
		ds, svc := setup(t)
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile, vars []fleet.FleetVarName) (*fleet.MDMAppleConfigProfile, error) {
			require.Equal(t, &teamID, p.TeamID)
			require.Equal(t, p.Identifier, mobileconfig.FleetFileVaultPayloadIdentifier)
			require.Equal(t, p.Name, mdm.FleetFileVaultProfileName)
			require.Contains(t, string(p.Mobileconfig), `MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUxEzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhvcml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoMClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VHcJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/Ufa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JSkPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVjE26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSwVeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTfUNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSndAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0HMOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLXKLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6ClieTTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=`)

			testPayload := getPayloadWithType(p.Mobileconfig, "com.apple.MCX.FileVault2")
			require.NotNil(t, testPayload)
			require.Equal(t, true, testPayload["Defer"])
			require.EqualValues(t, 0, testPayload["DeferForceAtUserLoginMaxBypassAttempts"])

			return nil, nil
		}

		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, ptr.Uint(teamID))
		require.NoError(t, err)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
	})
}

func TestMDMAppleDisableFileVaultAndEscrow(t *testing.T) {
	var wantTeamID uint
	ds, svc := setup(t)
	ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
		require.NotNil(t, teamID)
		require.Equal(t, wantTeamID, *teamID)
		require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, profileIdentifier)
		return nil
	}

	err := svc.MDMAppleDisableFileVaultAndEscrow(context.Background(), ptr.Uint(wantTeamID))
	require.NoError(t, err)
	require.True(t, ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFuncInvoked)
}

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

func TestClearHostPasscode(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	commander := &mockMDMAppleCommander{}
	activitySvc := &minimalMockFleetService{}
	svc := Service{ds: ds, authz: authorizer, mdmAppleCommander: commander, Service: activitySvc}

	iosHost := &fleet.Host{
		ID:       1,
		UUID:     "ios-host-uuid",
		Platform: "ios",
	}
	enrolledAuto := "On (automatic)"
	iosHost.MDM.EnrollmentStatus = &enrolledAuto

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return iosHost, nil
	}
	ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
		return true, nil
	}
	ds.GetMDMAppleDeviceUnlockTokenFunc = func(ctx context.Context, hostUUID string) ([]byte, error) {
		return []byte("unlock-token"), nil
	}

	t.Run("auth", func(t *testing.T) {
		cases := []struct {
			desc          string
			user          *fleet.User
			shouldFailAuth bool
		}{
			{"no role", test.UserNoRoles, true},
			{"global admin", test.UserAdmin, false},
			{"global maintainer", test.UserMaintainer, false},
			{"global observer", test.UserObserver, true},
			{"global observer+", test.UserObserverPlus, true},
			{"global technician", test.UserTechnician, true},
			// GitOps cannot list hosts (first auth check), so this fails.
			{"global gitops", test.UserGitOps, true},
			// Team-scoped users can list hosts but cannot write global MDM commands.
			{"team admin team1", test.UserTeamAdminTeam1, true},
			{"team maintainer team1", test.UserTeamMaintainerTeam1, true},
			{"team observer team1", test.UserTeamObserverTeam1, true},
			{"team observer+ team1", test.UserTeamObserverPlusTeam1, true},
		}
		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				ctx := test.UserContext(t.Context(), c.user)
				err := svc.ClearHostPasscode(ctx, iosHost.ID)
				checkAuthErr(t, c.shouldFailAuth, err)
			})
		}
	})

	t.Run("non-iOS platform fails", func(t *testing.T) {
		macHost := &fleet.Host{ID: 2, UUID: "mac-uuid", Platform: "darwin"}
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return macHost, nil
		}
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, macHost.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "only supported for iOS and iPadOS")
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return iosHost, nil
		}
	})

	t.Run("personal enrollment fails", func(t *testing.T) {
		personal := "On (personal)"
		host := &fleet.Host{ID: 3, UUID: "ios-personal", Platform: "ios"}
		host.MDM.EnrollmentStatus = &personal
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return host, nil
		}
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, host.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "personal device")
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return iosHost, nil
		}
	})

	t.Run("manual enrollment fails", func(t *testing.T) {
		manual := "On (manual)"
		host := &fleet.Host{ID: 4, UUID: "ios-manual", Platform: "ios"}
		host.MDM.EnrollmentStatus = &manual
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return host, nil
		}
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, host.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "manually enrolled")
		ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return iosHost, nil
		}
	})

	t.Run("not connected to Fleet MDM fails", func(t *testing.T) {
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return false, nil
		}
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, iosHost.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "doesn't have MDM turned on")
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
	})

	t.Run("missing unlock token returns error", func(t *testing.T) {
		ds.GetMDMAppleDeviceUnlockTokenFunc = func(ctx context.Context, hostUUID string) ([]byte, error) {
			return nil, nil
		}
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, iosHost.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unlock token is not yet available")
		ds.GetMDMAppleDeviceUnlockTokenFunc = func(ctx context.Context, hostUUID string) ([]byte, error) {
			return []byte("unlock-token"), nil
		}
	})

	t.Run("happy path", func(t *testing.T) {
		var commanderCalled bool
		commander.clearPasscodeFunc = func(ctx context.Context, host *fleet.Host, commandUUID string, unlockToken []byte) error {
			commanderCalled = true
			require.Equal(t, iosHost.UUID, host.UUID)
			require.Equal(t, []byte("unlock-token"), unlockToken)
			return nil
		}
		activitySvc.newActivityCalled = false
		ctx := test.UserContext(t.Context(), test.UserAdmin)
		err := svc.ClearHostPasscode(ctx, iosHost.ID)
		require.NoError(t, err)
		require.True(t, commanderCalled)
		require.True(t, activitySvc.newActivityCalled)
	})
}

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
