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
