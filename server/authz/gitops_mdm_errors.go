package authz

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GitOpsMDMCommandError replaces err, a failed MDM command write authorization, with a Fleet 5 migration message when the
// viewer has the GitOps role globally or on the fleet of teamID (nil for hosts in no fleet).
func GitOpsMDMCommandError(ctx context.Context, teamID *uint, err error) error {
	return gitOpsMDMCommandError(ctx, err, func(id uint) bool { return teamID != nil && id == *teamID })
}

// GitOpsHostListError is GitOpsMDMCommandError for the host list check made before an MDM command's target host is looked
// up, so GitOps users get the message without learning whether the host exists. A user who can't list hosts holds only
// GitOps roles, if any.
func GitOpsHostListError(ctx context.Context, err error) error {
	return gitOpsMDMCommandError(ctx, err, func(uint) bool { return true })
}

func gitOpsMDMCommandError(ctx context.Context, err error, matchesTeam func(teamID uint) bool) error {
	user := UserFromContext(ctx)
	if user == nil {
		return err
	}
	isGitOps := user.GlobalRole != nil && *user.GlobalRole == fleet.RoleGitOps
	for _, team := range user.Teams {
		if team.Role == fleet.RoleGitOps && matchesTeam(team.ID) {
			isGitOps = true
		}
	}
	if !isGitOps {
		return err
	}
	return &Forbidden{message: fleet.GitOpsMDMCommandErrorMessage, internal: err.Error(), subject: user}
}
