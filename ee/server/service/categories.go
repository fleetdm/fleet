package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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
		return nil, fleet.NewInvalidArgumentError("name", "is required")
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareCategory{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	category, err := svc.ds.NewSoftwareCategory(ctx, teamID, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new software category")
	}
	return category, nil
}

func (svc *Service) UpdateSoftwareCategory(ctx context.Context, id uint, name string) (*fleet.SoftwareCategory, error) {
	if name == "" {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("name", "is required")
	}

	// we need to get the category first to find its fleet id for authorization
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
	return updated, nil
}

func (svc *Service) DeleteSoftwareCategory(ctx context.Context, id uint) error {
	// we need to get the category first to find its fleet id for authorization
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
	return nil
}
