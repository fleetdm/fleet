package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListSelfServiceCategories(ctx context.Context, fleetID uint) ([]*fleet.SelfServiceCategory, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SelfServiceCategory{FleetID: fleetID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	categories, err := svc.ds.ListSelfServiceCategories(ctx, fleetID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list self-service categories")
	}
	return categories, nil
}

func (svc *Service) NewSelfServiceCategory(ctx context.Context, fleetID uint, name string) (*fleet.SelfServiceCategory, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SelfServiceCategory{FleetID: fleetID}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	category, err := svc.ds.NewSelfServiceCategory(ctx, fleetID, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new self-service category")
	}
	return category, nil
}

func (svc *Service) UpdateSelfServiceCategory(ctx context.Context, id uint, name string) (*fleet.SelfServiceCategory, error) {
	// we need to get the category first to find its fleet id for authorization
	category, err := svc.ds.SelfServiceCategory(ctx, id)
	if err != nil {
		if fleet.IsNotFound(err) {
			if authzErr := svc.authz.Authorize(ctx, &fleet.SelfServiceCategory{}, fleet.ActionWrite); authzErr != nil {
				return nil, authzErr
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get self-service category")
	}
	if err := svc.authz.Authorize(ctx, category, fleet.ActionWrite); err != nil {
		return nil, err
	}

	updated, err := svc.ds.UpdateSelfServiceCategory(ctx, id, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "update self-service category")
	}
	return updated, nil
}

func (svc *Service) DeleteSelfServiceCategory(ctx context.Context, id uint) error {
	// we need to get the category first to find its fleet id for authorization
	category, err := svc.ds.SelfServiceCategory(ctx, id)
	if err != nil {
		if fleet.IsNotFound(err) {
			if authzErr := svc.authz.Authorize(ctx, &fleet.SelfServiceCategory{}, fleet.ActionWrite); authzErr != nil {
				return authzErr
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "get self-service category")
	}

	if err := svc.authz.Authorize(ctx, category, fleet.ActionWrite); err != nil {
		return err
	}
	if err := svc.ds.DeleteSelfServiceCategory(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err, "delete self-service category")
	}
	return nil
}
