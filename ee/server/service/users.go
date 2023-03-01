package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// GetSSOUser is the premium implementation of svc.GetSSOUser, it allows to
// create users during the SSO flow the first time they log in if
// config.SSOSettings.EnableJITProvisioning is `true`
func (svc *Service) GetSSOUser(ctx context.Context, auth fleet.Auth) (*fleet.User, error) {
	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	// despite the fact that svc.NewUser will also validate the
	// email, we do it here to avoid hitting the database early if
	// the email happens to be invalid.
	if err := fleet.ValidateEmail(auth.UserID()); err != nil {
		return nil, ctxerr.New(ctx, "validating SSO response")
	}

	user, err := svc.Service.GetSSOUser(ctx, auth)
	var nfe fleet.NotFoundError
	switch {
	case err == nil:
		// If the user exists, we want to update the user roles from the attributes received
		// in the SAMLResponse.

		// If JIT provisioning is disabled, then we don't attempt to change the role of the user.
		if !config.SSOSettings.EnableJITProvisioning {
			return user, nil
		}

		newGlobalRole, newTeamsRoles, err := svc.userRolesFromSSOAttributes(ctx, auth)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "user roles from SSO attributes")
		}
		oldGlobalRole := user.GlobalRole
		oldTeamRoles := user.Teams

		// rolesChanged assumes that there cannot be multiple role entries for the same team,
		// which is ok because the "old" values comes from the database and the "new" values
		// come from fleet.RolesFromSSOAttributes which already checks for duplicates.
		if !rolesChanged(oldGlobalRole, oldTeamRoles, newGlobalRole, newTeamsRoles) {
			// Roles haven't changed, so nothing to do.
			return user, nil
		}

		user.GlobalRole = newGlobalRole
		user.Teams = newTeamsRoles

		err = svc.ds.SaveUser(ctx, user)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "save user")
		}
		if err := fleet.LogRoleChangeActivities(ctx, svc.ds, user, oldGlobalRole, oldTeamRoles, user); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "log activities for role change")
		}
		return user, nil
	case errors.As(err, &nfe):
		if !config.SSOSettings.EnableJITProvisioning {
			return nil, err
		}
	default:
		return nil, err
	}

	displayName := auth.UserDisplayName()
	if displayName == "" {
		displayName = auth.UserID()
	}

	// Retrieve (if set) user roles from SAML custom attributes.
	globalRole, teamRoles, err := svc.userRolesFromSSOAttributes(ctx, auth)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "user roles from SSO attributes")
	}

	user, err = svc.Service.NewUser(ctx, fleet.UserPayload{
		Name:       &displayName,
		Email:      ptr.String(auth.UserID()),
		SSOEnabled: ptr.Bool(true),
		GlobalRole: globalRole,
		Teams:      &teamRoles,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating new SSO user")
	}
	if err := svc.ds.NewActivity(
		ctx,
		user,
		fleet.ActivityTypeUserAddedBySSO{},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for SSO user creation")
	}
	return user, nil
}

// rolesChanged checks whether there was any change between the old and new roles.
//
// rolesChanged assumes that there cannot be multiple role entries for the same team.
func rolesChanged(oldGlobal *string, oldTeams []fleet.UserTeam, newGlobal *string, newTeams []fleet.UserTeam) bool {
	if (newGlobal != nil && (oldGlobal == nil || *oldGlobal != *newGlobal)) || (newGlobal == nil && oldGlobal != nil) {
		return true
	}
	if len(oldTeams) != len(newTeams) {
		return true
	}
	oldTeamsMap := make(map[uint]fleet.UserTeam)
	for _, oldTeam := range oldTeams {
		oldTeamsMap[oldTeam.Team.ID] = oldTeam
	}
	for _, newTeam := range newTeams {
		oldTeam, ok := oldTeamsMap[newTeam.Team.ID]
		if !ok {
			return true
		}
		if oldTeam.Role != newTeam.Role {
			return true
		}
	}
	return false
}

// userRolesFromSSOAttributes loads the global or team roles from custom SSO attributes.
//
// The returned `globalRole` and `teamRoles` are ready to be assigned to
// `fleet.User` struct fields `GlobalRole` and `Teams` respectively.
//
// If the custom attributes are not found, then the default global observer ("observer", nil, nil) is returned.
func (svc *Service) userRolesFromSSOAttributes(ctx context.Context, auth fleet.Auth) (globalRole *string, teamsRoles []fleet.UserTeam, err error) {
	ssoRolesInfo, err := fleet.RolesFromSSOAttributes(auth.AssertionAttributes())
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "invalid SSO attributes")
	}

	for _, teamRole := range ssoRolesInfo.Teams {
		team, err := svc.ds.Team(ctx, teamRole.ID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "invalid team")
		}
		teamsRoles = append(teamsRoles, fleet.UserTeam{
			Team: *team,
			Role: teamRole.Role,
		})
	}

	return ssoRolesInfo.Global, teamsRoles, nil
}
