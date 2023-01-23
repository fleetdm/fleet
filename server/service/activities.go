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
	Meta       *fleet.PaginationMetadata `json:"meta"`
	Activities []*fleet.Activity         `json:"activities"`
	Err        error                     `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listActivitiesRequest)
	activities, metadata, err := svc.ListActivities(ctx, fleet.ListActivitiesOptions{
		ListOptions: req.ListOptions,
	})
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return listActivitiesResponse{Meta: metadata, Activities: activities}, nil
}

// ListActivities returns a slice of activities for the whole organization
func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}
	return svc.ds.ListActivities(ctx, opt)
}

// logRoleChangeActivities stores the activities for role changes, globally and in teams.
func logRoleChangeActivities(ctx context.Context, ds fleet.Datastore, adminUser *fleet.User, oldRole *string, oldTeams []fleet.UserTeam, user *fleet.User) error {
	if user.GlobalRole != nil && (oldRole == nil || *oldRole != *user.GlobalRole) {
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeChangedUserGlobalRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      *user.GlobalRole,
			},
		); err != nil {
			return err
		}
	}
	if user.GlobalRole == nil && oldRole != nil {
		if err := ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeDeletedUserGlobalRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				OldRole:   *oldRole,
			},
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
			fleet.ActivityTypeChangedUserTeamRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      t.Role,
				TeamID:    t.ID,
				TeamName:  t.Name,
			},
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
			fleet.ActivityTypeDeletedUserTeamRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      o.Role,
				TeamID:    o.ID,
				TeamName:  o.Name,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (svc *Service) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	return svc.ds.NewActivity(ctx, user, activity)
}
