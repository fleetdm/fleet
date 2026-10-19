package service

import (
	"context"
	"crypto/sha1" //nolint:gosec // CSP CAThumbprint is defined as the SHA-1 hash
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const conditionalAccessSCEPInstallPrefix = "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/"

// newConditionalAccessWindowsProfilesTestService returns a service whose datastore has everything the Windows
// profiles endpoint needs. Tests override individual mocks to exercise failures.
func newConditionalAccessWindowsProfilesTestService(t *testing.T, certPEM []byte, serverURL, secret string) (*mock.Store, fleet.Service, context.Context) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	cfg.Server.PrivateKey = "test-private-key"
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil)

	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessCACert: {Name: fleet.MDMAssetConditionalAccessCACert, Value: certPEM},
		}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: serverURL}}, nil
	}
	ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{{Secret: secret}}, nil
	}
	return ds, svc, test.UserContext(ctx, test.UserAdmin)
}

// xmlText decodes the raw inner XML of an element into its text.
func xmlText(t *testing.T, raw string) string {
	t.Helper()
	var s string
	require.NoError(t, xml.Unmarshal([]byte("<x>"+raw+"</x>"), &s))
	return s
}

// syncMLTargets maps each LocURI of a parsed profile to its decoded <Data>, and returns the verbs in order.
func syncMLTargets(t *testing.T, profile string) (map[string]string, []string) {
	t.Helper()
	cmds, err := fleet.UnmarshallMultiTopLevelXMLProfile([]byte(profile))
	require.NoError(t, err)
	data := map[string]string{}
	var verbs []string
	for _, cmd := range cmds {
		verbs = append(verbs, cmd.XMLName.Local)
		data[cmd.GetTargetURI()] = xmlText(t, cmd.GetTargetData())
	}
	return data, verbs
}

// autoSelectRuleFromPolicyData decodes a Policy CSP <Data> value down to the Chromium rule it carries.
func autoSelectRuleFromPolicyData(t *testing.T, decodedData, elementID string) map[string]any {
	t.Helper()
	var payload struct {
		Enabled *struct{} `xml:"enabled"`
		Data    struct {
			ID    string `xml:"id,attr"`
			Value string `xml:"value,attr"`
		} `xml:"data"`
	}
	require.NoError(t, xml.Unmarshal([]byte("<x>"+decodedData+"</x>"), &payload))
	require.NotNil(t, payload.Enabled)
	require.Equal(t, elementID, payload.Data.ID)
	var rule map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload.Data.Value), &rule))
	return rule
}

func TestConditionalAccessGetIdPWindowsProfilesAuth(t *testing.T) {
	t.Parallel()
	_, svc, ctx := newConditionalAccessWindowsProfilesTestService(t, generateTestCertPEM(t), "https://fleet.example.com", "test-secret-123")

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"global admin", test.UserAdmin, false},
		{"global maintainer", test.UserMaintainer, false},
		{"global observer", test.UserObserver, true},
		{"global observer+", test.UserObserverPlus, true},
		{"global gitops", test.UserGitOps, false},
		{"global technician", test.UserTechnician, false},
		{"team admin", test.UserTeamAdminTeam1, true},
		{"team maintainer", test.UserTeamMaintainerTeam1, true},
		{"team observer", test.UserTeamObserverTeam1, true},
		{"team observer+", test.UserTeamObserverPlusTeam1, true},
		{"team gitops", test.UserTeamGitOpsTeam1, true},
		{"team technician", test.UserTeamTechnicianTeam1, true},
		{"user no roles", test.UserNoRoles, true},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(test.UserContext(ctx, tt.user))
			if tt.shouldFail {
				var forbiddenError *authz.Forbidden
				require.ErrorAs(t, err, &forbiddenError)
				require.Nil(t, profiles)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, profiles.SCEPProfile)
			require.NotEmpty(t, profiles.BrowserProfile)
		})
	}
}

func TestConditionalAccessGetIdPWindowsProfiles(t *testing.T) {
	t.Parallel()
	certPEM := generateTestCertPEM(t)
	block, _ := pem.Decode(certPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	thumbprint := sha1.Sum(caCert.Raw) //nolint:gosec // CSP CAThumbprint is defined as the SHA-1 hash

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		_, svc, ctx := newConditionalAccessWindowsProfilesTestService(t, certPEM, "https://fleet.example.com:8080", "AbC+/12=")
		profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(ctx)
		require.NoError(t, err)

		// SCEP profile: SCEP-only, user-scoped, one Exec on Install/Enroll.
		scepData, scepVerbs := syncMLTargets(t, profiles.SCEPProfile)
		require.Equal(t, 1, strings.Count(strings.Join(scepVerbs, ","), fleet.CmdExec))
		require.Equal(t, fleet.CmdExec, scepVerbs[len(scepVerbs)-1])
		require.Contains(t, scepData, conditionalAccessSCEPInstallPrefix+"Enroll")
		for loc := range scepData {
			require.True(t, strings.HasPrefix(loc, "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/"), loc)
		}
		scepProfile := &fleet.MDMWindowsConfigProfile{Name: "Okta SCEP", SyncML: []byte(profiles.SCEPProfile)}
		require.NoError(t, scepProfile.ValidateUserProvided(false))

		require.Equal(t, "$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL", scepData[conditionalAccessSCEPInstallPrefix+"ServerURL"])
		require.Equal(t, "$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_CHALLENGE", scepData[conditionalAccessSCEPInstallPrefix+"Challenge"])
		require.Equal(t, strings.ToUpper(hex.EncodeToString(thumbprint[:])), scepData[conditionalAccessSCEPInstallPrefix+"CAThumbprint"])
		require.Equal(t, "CN=Fleet conditional access for Okta,OU=$FLEET_VAR_CERTIFICATE_RENEWAL_ID",
			scepData[conditionalAccessSCEPInstallPrefix+"SubjectName"])
		require.Equal(t, "7+urn:device:fleet:uuid:$FLEET_VAR_HOST_UUID", scepData[conditionalAccessSCEPInstallPrefix+"SubjectAlternativeNames"])
		require.Equal(t, "1.3.6.1.5.5.7.3.2", scepData[conditionalAccessSCEPInstallPrefix+"EKUMapping"])
		require.Equal(t, "160", scepData[conditionalAccessSCEPInstallPrefix+"KeyUsage"])
		require.Equal(t, "2048", scepData[conditionalAccessSCEPInstallPrefix+"KeyLength"])
		require.Equal(t, "SHA-2", scepData[conditionalAccessSCEPInstallPrefix+"HashAlgorithm"])
		require.Equal(t, "2", scepData[conditionalAccessSCEPInstallPrefix+"KeyProtection"])
		require.NotContains(t, profiles.SCEPProfile, "AbC+/12=")

		foundVars, err := validateWindowsProfileFleetVariables(profiles.SCEPProfile, &fleet.LicenseInfo{Tier: fleet.TierPremium}, nil)
		require.NoError(t, err)
		slices.Sort(foundVars)
		require.Equal(t, []string{
			string(fleet.FleetVarCertificateRenewalID),
			string(fleet.FleetVarConditionalAccessSCEPChallenge),
			string(fleet.FleetVarConditionalAccessSCEPURL),
			string(fleet.FleetVarHostUUID),
			string(fleet.FleetVarSCEPWindowsCertificateID),
		}, foundVars)

		// Browser profile: device-scoped, no Exec, ADMX ingested before the policies are set.
		browserData, browserVerbs := syncMLTargets(t, profiles.BrowserProfile)
		require.Equal(t, []string{fleet.CmdAdd, fleet.CmdReplace, fleet.CmdReplace}, browserVerbs)
		browserProfile := &fleet.MDMWindowsConfigProfile{Name: "Okta browsers", SyncML: []byte(profiles.BrowserProfile)}
		require.NoError(t, browserProfile.ValidateUserProvided(false))
		foundVars, err = validateWindowsProfileFleetVariables(profiles.BrowserProfile, nil, nil)
		require.NoError(t, err)
		require.Empty(t, foundVars)

		admx := browserData["./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetConditionalAccess/Policy/FleetConditionalAccessAdmx"]
		require.Equal(t, conditionalAccessWindowsADMX, admx)

		wantRule := map[string]any{
			"pattern": "https://okta.fleet.example.com:8080",
			"filter":  map[string]any{"SUBJECT": map[string]any{"CN": fleet.ConditionalAccessOktaCertificateCN}},
		}
		for browser, uri := range map[string]string{
			"Chrome": "./Device/Vendor/MSFT/Policy/Config/FleetConditionalAccess~Policy~fleetca/ChromeAutoSelectCertificate",
			"Edge":   "./Device/Vendor/MSFT/Policy/Config/FleetConditionalAccess~Policy~fleetca/EdgeAutoSelectCertificate",
		} {
			require.Contains(t, browserData, uri)
			require.Equal(t, wantRule, autoSelectRuleFromPolicyData(t, browserData[uri], browser+"AutoSelectCertificate_Value"), browser)
		}
	})

	t.Run("enroll secret Windows can't send as a SCEP challenge", func(t *testing.T) {
		t.Parallel()
		_, svc, ctx := newConditionalAccessWindowsProfilesTestService(t, certPEM, "https://fleet.example.com", "bad_secret")
		profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(ctx)
		var badReqErr *fleet.BadRequestError
		require.ErrorAs(t, err, &badReqErr)
		require.Contains(t, err.Error(), "Allowed: letters, numbers, spaces, and ' ( ) + , - . / : = ?")
		require.Nil(t, profiles)
	})

	t.Run("missing server private key", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		cfg := config.TestConfig()
		cfg.Server.PrivateKey = ""
		svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil)
		profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(test.UserContext(ctx, test.UserAdmin))
		var badReqErr *fleet.BadRequestError
		require.ErrorAs(t, err, &badReqErr)
		require.Contains(t, err.Error(), "Fleet server private key is not configured")
		require.Nil(t, profiles)
	})

	invalidDER := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid DER data")})
	errorCases := []struct {
		name       string
		setup      func(ds *mock.Store)
		badRequest bool
		wantErr    string
	}{
		{
			name: "missing CA certificate",
			setup: func(ds *mock.Store) {
				ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
					return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
				}
			},
			wantErr: "conditional access CA certificate not configured",
		},
		{
			name: "invalid PEM certificate",
			setup: func(ds *mock.Store) {
				ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
					return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
						fleet.MDMAssetConditionalAccessCACert: {Name: fleet.MDMAssetConditionalAccessCACert, Value: []byte("not a valid PEM")},
					}, nil
				}
			},
			wantErr: "failed to decode CA certificate PEM",
		},
		{
			name: "invalid DER certificate",
			setup: func(ds *mock.Store) {
				ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
					return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
						fleet.MDMAssetConditionalAccessCACert: {Name: fleet.MDMAssetConditionalAccessCACert, Value: invalidDER},
					}, nil
				}
			},
			wantErr: "failed to parse CA certificate",
		},
		{
			name: "no enroll secrets",
			setup: func(ds *mock.Store) {
				ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
					return []*fleet.EnrollSecret{}, nil
				}
			},
			badRequest: true,
			wantErr:    "global enroll secret is not configured",
		},
		{
			name: "server URL not configured",
			setup: func(ds *mock.Store) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{}, nil
				}
			},
			badRequest: true,
			wantErr:    "server URL is not configured",
		},
		{
			name: "invalid server URL",
			setup: func(ds *mock.Store) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "://invalid-url"}}, nil
				}
			},
			wantErr: "failed to parse server URL",
		},
	}
	for _, c := range errorCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ds, svc, ctx := newConditionalAccessWindowsProfilesTestService(t, certPEM, "https://fleet.example.com", "test-secret")
			c.setup(ds)
			profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(ctx)
			require.ErrorContains(t, err, c.wantErr)
			if c.badRequest {
				var badReqErr *fleet.BadRequestError
				require.ErrorAs(t, err, &badReqErr)
			}
			require.Nil(t, profiles)
		})
	}
}

// Not parallel: it sets a dev mode override.
func TestConditionalAccessGetIdPWindowsProfilesEscaping(t *testing.T) {
	const mtlsURL = `https://okta.example.com/?a=<b>&c="d"`
	dev_mode.SetOverride("FLEET_DEV_OKTA_SSO_SERVER_URL", mtlsURL, t)

	_, svc, ctx := newConditionalAccessWindowsProfilesTestService(t, generateTestCertPEM(t), "https://fleet.example.com", "test-secret")
	profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(ctx)
	require.NoError(t, err)

	browserData, _ := syncMLTargets(t, profiles.BrowserProfile)
	rule := autoSelectRuleFromPolicyData(t,
		browserData["./Device/Vendor/MSFT/Policy/Config/FleetConditionalAccess~Policy~fleetca/ChromeAutoSelectCertificate"],
		"ChromeAutoSelectCertificate_Value")
	require.Equal(t, mtlsURL, rule["pattern"])
	require.NoError(t, (&fleet.MDMWindowsConfigProfile{Name: "Okta browsers", SyncML: []byte(profiles.BrowserProfile)}).ValidateUserProvided(false))
}
