package authz

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GitOpsMDMCommandError replaces err, a failed MDM command write authorization, with a Fleet 5 migration message when the
// viewer has the GitOps role globally or on the fleet of teamID (nil for hosts in no fleet).
func GitOpsMDMCommandError(ctx context.Context, teamID *uint, err error) error {
	return GitOpsFleetError(ctx, teamID, fleet.GitOpsMDMCommandErrorMessage, err)
}

// GitOpsHostListError is GitOpsMDMCommandError for the host list check made before an MDM command's target host is looked
// up, so GitOps users get the message without learning whether the host exists. A user who can't list hosts holds only
// GitOps roles, if any.
func GitOpsHostListError(ctx context.Context, err error) error {
	return GitOpsListError(ctx, fleet.GitOpsMDMCommandErrorMessage, err)
}

// GitOpsFleetError replaces err, a failed authorization, with message when the viewer has the GitOps role globally or on
// the fleet of teamID (nil for hosts in no fleet).
func GitOpsFleetError(ctx context.Context, teamID *uint, message string, err error) error {
	return gitOpsError(ctx, err, message, func(id uint) bool { return teamID != nil && id == *teamID })
}

// GitOpsListError is GitOpsFleetError for the host list check made before a host is looked up, so GitOps users get the
// message without learning whether the host exists. A user who can't list hosts holds only GitOps roles, if any.
func GitOpsListError(ctx context.Context, message string, err error) error {
	return gitOpsError(ctx, err, message, func(uint) bool { return true })
}

func gitOpsError(ctx context.Context, err error, message string, matchesTeam func(teamID uint) bool) error {
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
	forbidden := &Forbidden{message: message, internal: err.Error(), subject: user}
	// Keep the denial's details for the log.
	if authzErr, ok := errors.AsType[*Forbidden](err); ok {
		forbidden.internal, forbidden.object, forbidden.action = authzErr.internal, authzErr.object, authzErr.action
	}
	return forbidden
}
