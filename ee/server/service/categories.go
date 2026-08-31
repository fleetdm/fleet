package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) ListSoftwareCategories(ctx context.Context, teamID *uint) ([]fleet.SoftwareCategory, error) {
	if teamID == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("fleet_id", "fleet_id is required")
	}
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{TeamID: *teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	if *teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking if fleet exists")
		}
		if !exists {
			return nil, fleet.NewInvalidArgumentError("fleet_id", fmt.Sprintf("fleet %d does not exist", *teamID)).WithStatus(http.StatusNotFound)
		}
	}
	categories, err := svc.ds.ListSoftwareCategories(ctx, *teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list software categories")
	}
	return categories, nil
}

func (svc *Service) ListSelfServiceSoftwareCategoriesForHost(ctx context.Context, host *fleet.Host) ([]fleet.SoftwareCategory, error) {
	teamID := ptr.ValOrZero(host.TeamID)

	categories, err := svc.ds.ListSoftwareCategories(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list self-service software categories for host")
	}
	return categories, nil
}

func (svc *Service) NewSoftwareCategory(ctx context.Context, teamID *uint, name string) (*fleet.SoftwareCategory, error) {
	name = strings.TrimSpace(name)
	if err := (fleet.SoftwareCategory{Name: name}).Validate(); err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "validating new software category")
	}
	if teamID == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("fleet_id", "fleet_id is required")
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{TeamID: *teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	if *teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking if fleet exists")
		}
		if !exists {
			return nil, fleet.NewInvalidArgumentError("fleet_id", fmt.Sprintf("fleet %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		}
	}

	category, err := svc.ds.NewSoftwareCategory(ctx, *teamID, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new software category")
	}

	teamName, err := svc.teamNameForActivity(ctx, category.TeamID)
	if err != nil {
		return nil, err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedSelfServiceCategory{
		SelfServiceCategoryName: category.Name,
		TeamID:                  new(category.TeamID),
		TeamName:                teamName,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for added self-service category")
	}
	return category, nil
}

func (svc *Service) UpdateSoftwareCategory(ctx context.Context, id uint, name string) (*fleet.SoftwareCategory, error) {
	name = strings.TrimSpace(name)
	if err := (fleet.SoftwareCategory{Name: name}).Validate(); err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "validating updated software category")
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

	if category.Name == name {
		return category, nil
	}

	updated, err := svc.ds.UpdateSoftwareCategory(ctx, id, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "update software category")
	}

	teamName, err := svc.teamNameForActivity(ctx, updated.TeamID)
	if err != nil {
		return nil, err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeEditedSelfServiceCategory{
		SelfServiceCategoryName: updated.Name,
		TeamID:                  new(updated.TeamID),
		TeamName:                teamName,
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

	teamName, err := svc.teamNameForActivity(ctx, category.TeamID)
	if err != nil {
		return err
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedSelfServiceCategory{
		SelfServiceCategoryName: category.Name,
		TeamID:                  new(category.TeamID),
		TeamName:                teamName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for deleted self-service category")
	}
	return nil
}

func (svc *Service) removeDuplicateOrMissingCategories(ctx context.Context, teamID uint, names []string) ([]string, []uint, error) {
	names = server.RemoveDuplicatesFromSlice(names)
	categories := []string{}
	ids := []uint{}
	if len(names) == 0 {
		return categories, ids, nil
	}
	categoryMap, err := svc.ds.GetSoftwareCategoryNameToIDMap(ctx, teamID, names)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "getting software category name to id map")
	}
	for _, name := range names {
		if id, ok := categoryMap[name]; ok {
			categories = append(categories, name)
			ids = append(ids, id)
		}
	}
	return categories, ids, nil
}

func (svc *Service) teamNameForActivity(ctx context.Context, teamID uint) (*string, error) {
	if teamID == 0 {
		return nil, nil
	}
	tm, err := svc.ds.TeamLite(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching team name for category activity")
	}
	return &tm.Name, nil
}

func trimAndValidateCategories(ctx context.Context, categories []string) error {
	for i, name := range categories {
		categories[i] = strings.TrimSpace(name)
		err := (fleet.SoftwareCategory{Name: categories[i]}).Validate()
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "category %q", categories[i])
		}
	}
	return nil
}
