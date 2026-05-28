package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// TODO(JK): the GitOps role is currently denied on every software_category
// action. Revisit when GitOps support for self-service categories lands —
// gitops will likely need read+write to manage fleet category lists from YAML.

func (svc *Service) ListSoftwareCategories(ctx context.Context, teamID uint) ([]*fleet.SoftwareCategory, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	categories, err := svc.ds.ListSoftwareCategories(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list software categories")
	}
	return categories, nil
}

func (svc *Service) NewSoftwareCategory(ctx context.Context, teamID uint, name string) (*fleet.SoftwareCategory, error) {
	if name == "" {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("name", "name is required")
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// teamID=0 is the "Unassigned" scope and doesn't need an existence check.
	if teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking if team exists")
		}
		if !exists {
			return nil, fleet.NewInvalidArgumentError("fleet_id", fmt.Sprintf("fleet %d does not exist", teamID)).
				WithStatus(http.StatusNotFound)
		}
	}

	category, err := svc.ds.NewSoftwareCategory(ctx, teamID, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new software category")
	}

	fleetName, err := svc.fleetNameForActivity(ctx, category.TeamID)
	if err != nil {
		return nil, err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedSelfServiceCategory{
		SelfServiceCategoryName: category.Name,
		FleetID:                 category.TeamID,
		FleetName:               fleetName,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for added self-service category")
	}
	return category, nil
}

func (svc *Service) UpdateSoftwareCategory(ctx context.Context, id uint, name string) (*fleet.SoftwareCategory, error) {
	if name == "" {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("name", "name is required")
	}

	// we need to load the category first to scope authz to its team_id
	category, err := svc.ds.SoftwareCategory(ctx, id)
	if err != nil {
		if fleet.IsNotFound(err) {
			if authzErr := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{}, fleet.ActionWrite); authzErr != nil {
				return nil, authzErr
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get software category")
	}
	if err := svc.authz.Authorize(ctx, category, fleet.ActionWrite); err != nil {
		return nil, err
	}

	updated, err := svc.ds.UpdateSoftwareCategory(ctx, id, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "update software category")
	}

	fleetName, err := svc.fleetNameForActivity(ctx, updated.TeamID)
	if err != nil {
		return nil, err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeEditedSelfServiceCategory{
		SelfServiceCategoryName: updated.Name,
		FleetID:                 updated.TeamID,
		FleetName:               fleetName,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for edited self-service category")
	}
	return updated, nil
}

func (svc *Service) DeleteSoftwareCategory(ctx context.Context, id uint) error {
	// we need to load the category first to scope authz to its team_id
	category, err := svc.ds.SoftwareCategory(ctx, id)
	if err != nil {
		if fleet.IsNotFound(err) {
			if authzErr := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{}, fleet.ActionWrite); authzErr != nil {
				return authzErr
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "get software category")
	}

	if err := svc.authz.Authorize(ctx, category, fleet.ActionWrite); err != nil {
		return err
	}
	if err := svc.ds.DeleteSoftwareCategory(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software category")
	}

	fleetName, err := svc.fleetNameForActivity(ctx, category.TeamID)
	if err != nil {
		return err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedSelfServiceCategory{
		SelfServiceCategoryName: category.Name,
		FleetID:                 category.TeamID,
		FleetName:               fleetName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for deleted self-service category")
	}
	return nil
}

// fleetNameForActivity returns the team's name for use in a category activity.
// fleet_id=0 ("Unassigned") has no associated team, so an empty string is returned.
func (svc *Service) fleetNameForActivity(ctx context.Context, teamID uint) (string, error) {
	if teamID == 0 {
		return "", nil
	}
	tm, err := svc.ds.TeamLite(ctx, teamID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "fetching fleet name for category activity")
	}
	return tm.Name, nil
}
