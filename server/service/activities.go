package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get activities
////////////////////////////////////////////////////////////////////////////////

type listActivitiesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listActivitiesResponse struct {
	Activities []*fleet.Activity `json:"activities"`
	Err        error             `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listActivitiesRequest)
	activities, err := svc.ListActivities(ctx, req.ListOptions)
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return listActivitiesResponse{Activities: activities}, nil
}

// ListActivities returns a slice of activities for the whole organization
func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Activity, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.ListActivities(ctx, opt)
}

// logRoleChangeActivities stores the activities for role changes, globally and in teams.
func logRoleChangeActivities(ctx context.Context, ds fleet.Datastore, adminUser *fleet.User, oldRole *string, oldTeams []fleet.UserTeam, user *fleet.User) error {
	if user.GlobalRole != nil && (oldRole == nil || *oldRole != *user.GlobalRole) {
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeChangedUserGlobalRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "role": *user.GlobalRole},
		); err != nil {
			return err
		}
	}
	if user.GlobalRole == nil && oldRole != nil {
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeDeletedUserGlobalRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "role": *oldRole},
		); err != nil {
			return err
		}
	}
	oldTeamsLookup := make(map[uint]fleet.UserTeam, len(oldTeams))
	for _, t := range oldTeams {
		oldTeamsLookup[t.ID] = t
	}

	newTeamLookup := make(map[uint]struct{}, len(user.Teams))
	for _, t := range user.Teams {
		newTeamLookup[t.ID] = struct{}{}
		o, ok := oldTeamsLookup[t.ID]
		if ok && o.Role == t.Role {
			continue
		}
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeChangedUserTeamRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "team_name": t.Name, "team_id": t.ID, "role": t.Role},
		); err != nil {
			return err
		}
	}
	for _, o := range oldTeams {
		if _, ok := newTeamLookup[o.ID]; ok {
			continue
		}
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeDeletedUserTeamRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "team_name": o.Name, "team_id": o.ID, "role": o.Role},
		); err != nil {
			return err
		}
	}
	return nil
}
