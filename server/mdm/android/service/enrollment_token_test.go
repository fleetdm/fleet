package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// An unauthenticated caller with an invalid enroll secret must get the same
// error whether or not Android MDM is configured, so the endpoint can't be
// used to probe configuration state.
func TestCreateEnrollmentTokenUniformInvalidSecretError(t *testing.T) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	fleetDS := InitCommonDSMocks()
	fleetDS.Store.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return nil, &notFoundError{}
	}
	svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
	require.NoError(t, err)

	badSecretErr := func(configured bool, idpSessionID string) error {
		fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.MDM.AndroidEnabledAndConfigured = configured
			return appCfg, nil
		}
		_, err := svc.CreateEnrollmentToken(t.Context(), "bogus-secret", idpSessionID, false)
		require.Error(t, err)
		return err
	}
	errNotConfigured := badSecretErr(false, "")
	errConfigured := badSecretErr(true, "")
	// The service has no idP session store, so this only returns the uniform
	// auth error if the secret is verified before idP session validation.
	errIdPSession := badSecretErr(true, "bogus-session")

	var authFailed *fleet.AuthFailedError
	require.ErrorAs(t, errNotConfigured, &authFailed)
	require.ErrorAs(t, errConfigured, &authFailed)
	require.ErrorAs(t, errIdPSession, &authFailed)
	require.Equal(t, errConfigured.Error(), errNotConfigured.Error())
	require.Equal(t, errConfigured.Error(), errIdPSession.Error())
}

// Enrollment tokens point at the enrolling fleet's default policy, so its passcode is set during setup.
func TestCreateEnrollmentTokenUsesFleetDefaultPolicy(t *testing.T) {
	const enterpriseName = "enterprises/LC00test"
	teamID := uint(4)

	cases := []struct {
		name           string
		secretTeamID   *uint
		checksums      map[uint][]byte
		patchErr       error
		wantPatched    []string
		wantPolicyName string
	}{
		{
			name:           "unassigned",
			wantPatched:    []string{enterpriseName + "/policies/fleet0"},
			wantPolicyName: enterpriseName + "/policies/fleet0",
		},
		{
			name:           "fleet",
			secretTeamID:   &teamID,
			wantPatched:    []string{enterpriseName + "/policies/fleet4"},
			wantPolicyName: enterpriseName + "/policies/fleet4",
		},
		{
			name:           "fleet policy already sent",
			secretTeamID:   &teamID,
			checksums:      map[uint][]byte{teamID: []byte("checksum")},
			wantPolicyName: enterpriseName + "/policies/fleet4",
		},
		{
			name:           "falls back to the global policy when the patch fails",
			secretTeamID:   &teamID,
			patchErr:       errors.New("quota"),
			wantPatched:    []string{enterpriseName + "/policies/fleet4"},
			wantPolicyName: enterpriseName + "/policies/1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			androidAPIClient := android_mock.Client{}
			androidAPIClient.InitCommonMocks()
			var patched []string
			androidAPIClient.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
				patched = append(patched, policyName)
				require.True(t, opts.ExcludeApps)
				require.NotNil(t, policy.StatusReportingSettings)
				return policy, c.patchErr
			}
			var tokenPolicyName string
			androidAPIClient.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, _ string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
				tokenPolicyName = token.PolicyName
				return &androidmanagement.EnrollmentToken{Value: "token"}, nil
			}

			fleetDS := InitCommonDSMocks()
			fleetDS.Store.VerifyEnrollSecretFunc = func(_ context.Context, _ string) (*fleet.EnrollSecret, error) {
				return &fleet.EnrollSecret{Secret: "secret", TeamID: c.secretTeamID}, nil
			}
			fleetDS.Store.TeamLiteFunc = func(_ context.Context, id uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: id}, nil
			}
			fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.MDM.AndroidEnabledAndConfigured = true
				return appCfg, nil
			}
			fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
				return &android.Enterprise{EnterpriseID: "LC00test"}, nil
			}
			fleetDS.Store.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{fleet.MDMAssetAndroidFleetServerSecret: {Value: []byte("secret")}}, nil
			}
			fleetDS.Store.GetAndroidFleetDefaultPolicyChecksumsFunc = func(_ context.Context) (map[uint][]byte, error) {
				return c.checksums, nil
			}
			fleetDS.Store.ListAndroidFleetDefaultPolicyProfilesFunc = func(_ context.Context, gotTeamID *uint) ([]*fleet.MDMAndroidConfigProfile, error) {
				require.NotNil(t, gotTeamID)
				return []*fleet.MDMAndroidConfigProfile{{Name: "passcode", TeamID: gotTeamID, RawJSON: []byte(`{"passwordPolicies": [{"passwordMinimumLength": 6}]}`)}}, nil
			}
			var savedTeamIDs []uint
			fleetDS.Store.SetAndroidFleetDefaultPolicyChecksumFunc = func(_ context.Context, gotTeamID uint, _ []byte) error {
				savedTeamIDs = append(savedTeamIDs, gotTeamID)
				return nil
			}
			svc, err := NewServiceWithClient(slog.New(slog.DiscardHandler), fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
			require.NoError(t, err)

			_, err = svc.CreateEnrollmentToken(t.Context(), "secret", "", true)
			require.NoError(t, err)
			require.Equal(t, c.wantPolicyName, tokenPolicyName)
			require.Equal(t, c.wantPatched, patched)
			if len(c.wantPatched) > 0 && c.patchErr == nil {
				var wantTeamID uint
				if c.secretTeamID != nil {
					wantTeamID = *c.secretTeamID
				}
				require.Equal(t, []uint{wantTeamID}, savedTeamIDs)
			} else {
				require.Empty(t, savedTeamIDs)
			}
		})
	}
}
