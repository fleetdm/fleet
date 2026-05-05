package fleet

import (
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestValidateUserRoles(t *testing.T) {
	checkErrCode := func(code int) func(error) bool {
		return func(err error) bool {
			errError, ok := err.(*Error)
			if !ok {
				return false
			}
			return errError.Code == code
		}
	}

	for _, tc := range []struct {
		name     string
		create   bool
		payload  UserPayload
		license  LicenseInfo
		checkErr func(err error) bool
	}{
		{
			name:   "global-gitops-create-not-premium",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    ptr.Bool(true),
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: func(err error) bool {
				return err == ErrMissingLicense
			},
		},
		{
			name:   "global-gitops-create-api-only",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    ptr.Bool(true),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: nil,
		},
		{
			name:   "global-gitops-create-not-api-only",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    ptr.Bool(false),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "global-gitops-create-api-only-not-set",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    nil,
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "global-gitops-create-api-only-not-set",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    nil,
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "global-gitops-modify-not-api-only",
			create: false,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    ptr.Bool(false),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "global-gitops-modify-api-only-not-set",
			create: false,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleGitOps),
				APIOnly:    nil,
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: nil,
		},
		{
			name:   "team-gitops-create-mixed-with-other-roles",
			create: true,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}, {Role: RoleMaintainer}},
				APIOnly: ptr.Bool(true),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: nil,
		},
		{
			name:   "team-gitops-modify-mixed-with-other-roles",
			create: false,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}, {Role: RoleMaintainer}},
				APIOnly: ptr.Bool(true),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: nil,
		},
		{
			name:   "team-gitops-create-api-only-false",
			create: true,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}},
				APIOnly: ptr.Bool(false),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "team-gitops-create-api-only-not-set",
			create: true,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}},
				APIOnly: nil,
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "team-gitops-modify-to-not-api-only",
			create: false,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}},
				APIOnly: ptr.Bool(false),
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: checkErrCode(ErrAPIOnlyRole),
		},
		{
			name:   "team-gitops-modify-api-only-not-set-should-succeed",
			create: false,
			payload: UserPayload{
				Teams:   &[]UserTeam{{Role: RoleGitOps}},
				APIOnly: nil, // not updating the APIOnly status.
			},
			license: LicenseInfo{
				Tier: TierPremium,
			},
			checkErr: nil,
		},
		{
			name:   "global-observer-modify-to-not-api-only",
			create: false,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleObserver),
				APIOnly:    ptr.Bool(false),
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: nil,
		},
		{
			name:   "global-invalid-role",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String("foobar"),
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: checkErrCode(ErrNoRoleNeeded),
		},
		{
			name:   "team-invalid-role",
			create: true,
			payload: UserPayload{
				Teams: &[]UserTeam{{Role: "foobar"}},
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: checkErrCode(ErrNoRoleNeeded),
		},
		{
			name:   "global-and-team-role-set",
			create: true,
			payload: UserPayload{
				GlobalRole: ptr.String(RoleObserver),
				Teams:      &[]UserTeam{{Role: RoleObserver}},
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: checkErrCode(ErrNoRoleNeeded),
		},
		{
			name:   "no-roles-set",
			create: true,
			payload: UserPayload{
				GlobalRole: nil,
				Teams:      &[]UserTeam{},
			},
			license: LicenseInfo{
				Tier: TierFree,
			},
			checkErr: checkErrCode(ErrNoRoleNeeded),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUserRoles(tc.create, tc.payload, tc.license)
			if err == nil {
				if tc.checkErr != nil {
					t.Errorf("expected an error: %+v", tc)
				}
			} else { // err != nil
				if tc.checkErr == nil {
					t.Errorf("unexpected error: %s %+v", err, tc)
				} else {
					require.True(t, tc.checkErr(err), "err_type=%T, err=%s", err, err)
				}
			}
		})
	}
}

func TestTeamMDMCopy(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var tm *TeamMDM
		require.Nil(t, tm.Copy())
	})

	t.Run("copy value fields", func(t *testing.T) {
		tm := &TeamMDM{
			EnableDiskEncryption: true,
			MacOSUpdates: AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("10.15.4"),
				Deadline:       optjson.SetString("2020-01-01"),
			},
			MacOSSetup: MacOSSetup{
				BootstrapPackage:            optjson.SetString("bootstrap"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("assistant"),
			},
		}
		clone := tm.Copy()
		require.NotNil(t, clone)
		require.NotSame(t, tm, clone)
		require.Equal(t, tm, clone)
	})

	t.Run("copy MacOSSettings", func(t *testing.T) {
		tm := &TeamMDM{
			MacOSSettings: MacOSSettings{
				CustomSettings:                 []MDMProfileSpec{{Path: "a"}, {Path: "b"}},
				DeprecatedEnableDiskEncryption: ptr.Bool(false),
			},
		}
		clone := tm.Copy()
		require.NotSame(t, tm, clone)
		require.Equal(t, tm, clone)
		require.NotEqual(t,
			reflect.ValueOf(tm.MacOSSettings.CustomSettings).Pointer(),
			reflect.ValueOf(clone.MacOSSettings.CustomSettings).Pointer(),
		)
		require.NotSame(t, tm.MacOSSettings.DeprecatedEnableDiskEncryption, clone.MacOSSettings.DeprecatedEnableDiskEncryption)
	})
}

func TestTeamConfigCopy(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var tc *TeamConfig
		require.Nil(t, tc.Copy())
	})

	t.Run("deep copy webhook settings", func(t *testing.T) {
		tc := &TeamConfig{
			WebhookSettings: TeamWebhookSettings{
				HostStatusWebhook: &HostStatusWebhookSettings{
					Enable:         true,
					DestinationURL: "https://example.com",
					HostPercentage: 0.5,
					DaysCount:      7,
				},
				FailingPoliciesWebhook: FailingPoliciesWebhookSettings{
					Enable:         true,
					DestinationURL: "https://policies.example.com",
					PolicyIDs:      []uint{1, 2, 3},
					HostBatchSize:  100,
				},
			},
		}

		clone := tc.Copy()
		require.NotNil(t, clone)
		require.NotSame(t, tc, clone)

		// Verify deep copy of HostStatusWebhook pointer
		require.NotSame(t, tc.WebhookSettings.HostStatusWebhook, clone.WebhookSettings.HostStatusWebhook)
		require.Equal(t, tc.WebhookSettings.HostStatusWebhook, clone.WebhookSettings.HostStatusWebhook)

		// Verify deep copy of PolicyIDs slice
		require.NotEqual(t,
			reflect.ValueOf(tc.WebhookSettings.FailingPoliciesWebhook.PolicyIDs).Pointer(),
			reflect.ValueOf(clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs).Pointer(),
		)
		require.Equal(t, tc.WebhookSettings.FailingPoliciesWebhook.PolicyIDs, clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)

		// Modify original and verify clone is unaffected
		tc.WebhookSettings.HostStatusWebhook.Enable = false
		tc.WebhookSettings.FailingPoliciesWebhook.PolicyIDs[0] = 999
		require.True(t, clone.WebhookSettings.HostStatusWebhook.Enable)
		require.Equal(t, uint(1), clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs[0])
	})

	t.Run("deep copy features", func(t *testing.T) {
		tc := &TeamConfig{
			Features: Features{
				EnableHostUsers:         true,
				EnableSoftwareInventory: true,
				AdditionalQueries:       ptr.RawMessage([]byte(`{"query": "test"}`)),
				DetailQueryOverrides: map[string]*string{
					"key1": ptr.String("value1"),
					"key2": ptr.String("value2"),
				},
			},
		}

		clone := tc.Copy()
		require.NotNil(t, clone)
		require.NotSame(t, tc, clone)

		// Verify deep copy of AdditionalQueries
		require.NotSame(t, tc.Features.AdditionalQueries, clone.Features.AdditionalQueries)
		require.Equal(t, tc.Features.AdditionalQueries, clone.Features.AdditionalQueries)

		// Verify deep copy of DetailQueryOverrides map
		require.NotEqual(t,
			reflect.ValueOf(tc.Features.DetailQueryOverrides).Pointer(),
			reflect.ValueOf(clone.Features.DetailQueryOverrides).Pointer(),
		)
		require.Equal(t, tc.Features.DetailQueryOverrides, clone.Features.DetailQueryOverrides)

		// Modify original and verify clone is unaffected
		tc.Features.EnableHostUsers = false
		*tc.Features.DetailQueryOverrides["key1"] = "modified"
		require.True(t, clone.Features.EnableHostUsers)
		require.Equal(t, "value1", *clone.Features.DetailQueryOverrides["key1"])
	})
}
