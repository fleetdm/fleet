package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestValidateUserRolesGitOps(t *testing.T) {
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
			checkErr: checkErrCode(ErrTeamGitOpsRoleMustBeUnique),
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
			checkErr: checkErrCode(ErrTeamGitOpsRoleMustBeUnique),
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
