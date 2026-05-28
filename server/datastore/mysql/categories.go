package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) ListSelfServiceCategories(ctx context.Context, fleetID uint) ([]*fleet.SelfServiceCategory, error) {
	return nil, nil
}

func (ds *Datastore) SelfServiceCategory(ctx context.Context, id uint) (*fleet.SelfServiceCategory, error) {
	return nil, ctxerr.Wrap(ctx, notFound("SelfServiceCategory").WithID(id))
}

func (ds *Datastore) NewSelfServiceCategory(ctx context.Context, fleetID uint, name string) (*fleet.SelfServiceCategory, error) {
	return nil, nil
}

func (ds *Datastore) UpdateSelfServiceCategory(ctx context.Context, id uint, name string) (*fleet.SelfServiceCategory, error) {
	return nil, nil
}

func (ds *Datastore) DeleteSelfServiceCategory(ctx context.Context, id uint) error {
	return nil
}
