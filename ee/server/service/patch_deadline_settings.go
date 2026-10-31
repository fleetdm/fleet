package service

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) GetPatchDeadlineSettings(ctx context.Context, teamID *uint) (*fleet.PatchDeadlineSettings, error) {
	if err := svc.authorizePatchDeadlineSettings(ctx, teamID, fleet.ActionRead); err != nil {
		return nil, err
	}
	settings, err := svc.ds.GetPatchDeadlineSettings(ctx, *teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get patch deadline settings")
	}
	return settings, nil
}

func (svc *Service) UpdatePatchDeadlineSettings(ctx context.Context, teamID *uint, days, tm any) (*fleet.PatchDeadlineSettings, error) {
	if err := svc.authorizePatchDeadlineSettings(ctx, teamID, fleet.ActionWrite); err != nil {
		return nil, err
	}

	validDays, validTime, err := fleet.ValidatePatchDeadlineSettings(days, tm)
	if err != nil {
		return nil, err
	}

	ctx = ctxdb.RequirePrimary(ctx, true)
	current, err := svc.ds.GetPatchDeadlineSettings(ctx, *teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get patch deadline settings")
	}
	if slices.Equal(current.Days, validDays) && current.Time == validTime {
		return current, nil
	}

	updated := fleet.PatchDeadlineSettings{TeamID: *teamID, Days: validDays, Time: validTime}
	if err := svc.ds.SetPatchDeadlineSettings(ctx, updated); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "set patch deadline settings")
	}

	teamName, err := svc.teamNameForActivity(ctx, *teamID)
	if err != nil {
		return nil, err
	}
	activity := fleet.ActivityTypeEditedPatchDeadlineSettings{
		TeamID:   ptr.UintOrNilIfZero(*teamID),
		TeamName: teamName,
		Days:     validDays,
		Time:     validTime,
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for edited patch deadline settings")
	}
	return &updated, nil
}

// authorizePatchDeadlineSettings checks that teamID is present, authorized for action,
// and refers to an existing fleet.
func (svc *Service) authorizePatchDeadlineSettings(ctx context.Context, teamID *uint, action string) error {
	if teamID == nil {
		svc.authz.SkipAuthorization(ctx)
		return fleet.NewInvalidArgumentError("fleet_id", "fleet_id is required")
	}
	if err := svc.authz.Authorize(ctx, &fleet.PatchDeadlineSettings{TeamID: *teamID}, action); err != nil {
		return err
	}
	if *teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if fleet exists")
		}
		if !exists {
			return fleet.NewInvalidArgumentError("fleet_id", fmt.Sprintf("fleet %d does not exist", *teamID)).WithStatus(http.StatusNotFound)
		}
	}
	return nil
}
