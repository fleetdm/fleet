package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

//////////////////////////////////////////////////////////////////////////////////
// List custom host vitals
//////////////////////////////////////////////////////////////////////////////////

func listCustomHostVitalsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListCustomHostVitalsRequest)
	vitals, meta, count, err := svc.ListCustomHostVitals(ctx, req.ListOptions)
	return fleet.ListCustomHostVitalsResponse{
		CustomHostVitals: vitals,
		Meta:             meta,
		Count:            count,
		Err:              err,
	}, nil
}

func (svc *Service) ListCustomHostVitals(
	ctx context.Context,
	opts fleet.ListOptions,
) (customHostVitals []fleet.CustomHostVital, meta *fleet.PaginationMetadata, count int, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.CustomHostVital{}, fleet.ActionRead); err != nil {
		return nil, nil, 0, err
	}

	// Always include pagination info.
	opts.IncludeMetadata = true
	if opts.OrderKey == "" {
		opts.OrderKey = "name"
		opts.OrderDirection = fleet.OrderAscending
	}

	customHostVitals, meta, count, err = svc.ds.ListCustomHostVitals(ctx, opts)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "list custom host vitals")
	}
	return customHostVitals, meta, count, nil
}

//////////////////////////////////////////////////////////////////////////////////
// Create custom host vital
//////////////////////////////////////////////////////////////////////////////////

func createCustomHostVitalEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateCustomHostVitalRequest)
	vital, err := svc.CreateCustomHostVital(ctx, req.Name)
	if err != nil {
		return fleet.CreateCustomHostVitalResponse{Err: err}, nil
	}
	return fleet.CreateCustomHostVitalResponse{CustomHostVital: vital}, nil
}

func (svc *Service) CreateCustomHostVital(ctx context.Context, name string) (*fleet.CustomHostVital, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CustomHostVital{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := fleet.ValidateCustomHostVitalName(name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate custom host vital name")
	}

	vital, err := svc.ds.CreateCustomHostVital(ctx, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating custom host vital")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedCustomHostVital{
			CustomHostVitalID:   vital.ID,
			CustomHostVitalName: vital.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for custom host vital creation")
	}

	return &vital, nil
}

//////////////////////////////////////////////////////////////////////////////////
// Update (rename) custom host vital
//////////////////////////////////////////////////////////////////////////////////

func updateCustomHostVitalEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UpdateCustomHostVitalRequest)
	vital, err := svc.UpdateCustomHostVital(ctx, req.ID, req.Name)
	if err != nil {
		return fleet.UpdateCustomHostVitalResponse{Err: err}, nil
	}
	return fleet.UpdateCustomHostVitalResponse{CustomHostVital: vital}, nil
}

func (svc *Service) UpdateCustomHostVital(ctx context.Context, id uint, name string) (*fleet.CustomHostVital, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CustomHostVital{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := fleet.ValidateCustomHostVitalName(name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate custom host vital name")
	}

	vital, err := svc.ds.UpdateCustomHostVital(ctx, id, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating custom host vital")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedCustomHostVital{
			CustomHostVitalID:   vital.ID,
			CustomHostVitalName: vital.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for custom host vital edit")
	}

	return &vital, nil
}

//////////////////////////////////////////////////////////////////////////////////
// Delete custom host vital
//////////////////////////////////////////////////////////////////////////////////

func deleteCustomHostVitalEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteCustomHostVitalRequest)
	err := svc.DeleteCustomHostVital(ctx, req.ID)
	return fleet.DeleteCustomHostVitalResponse{Err: err}, nil
}

func (svc *Service) DeleteCustomHostVital(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.CustomHostVital{}, fleet.ActionWrite); err != nil {
		return err
	}

	name, err := svc.ds.DeleteCustomHostVital(ctx, id)
	if err != nil {
		if usedErr, ok := errors.AsType[*fleet.CustomHostVitalUsedError](err); ok {
			return ctxerr.Wrap(ctx, &fleet.ConflictError{
				Message: fmt.Sprintf("Couldn't delete. %s", usedErr.Error()),
			}, "delete custom host vital")
		}
		return ctxerr.Wrap(ctx, err, "delete custom host vital")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedCustomHostVital{
			CustomHostVitalID:   id,
			CustomHostVitalName: name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for custom host vital deletion")
	}

	return nil
}

//////////////////////////////////////////////////////////////////////////////////
// Set host custom host vital value
//////////////////////////////////////////////////////////////////////////////////

func setHostCustomHostVitalValueEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.SetHostCustomHostVitalValueRequest)
	err := svc.SetHostCustomHostVitalValue(ctx, req.HostID, req.ID, req.Value)
	return fleet.SetHostCustomHostVitalValueResponse{Err: err}, nil
}

func (svc *Service) SetHostCustomHostVitalValue(ctx context.Context, hostID uint, vitalID uint, value string) error {
	// Authorize against the host so team-scoped roles are enforced (host-write pattern).
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find host for setting custom host vital value")
	}

	if err := svc.authz.Authorize(ctx, &fleet.HostCustomHostVitalValue{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	vital, err := svc.customHostVitalByID(ctx, vitalID)
	if err != nil {
		return err
	}

	if err := svc.ds.SetHostCustomHostVitalValue(ctx, hostID, vitalID, value); err != nil {
		return ctxerr.Wrap(ctx, err, "set host custom host vital value")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedCustomHostVitalValue{
			HostID:              hostID,
			HostDisplayName:     host.DisplayName(),
			CustomHostVitalID:   vitalID,
			CustomHostVitalName: vital.Name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for custom host vital value edit")
	}

	return nil
}

func (svc *Service) customHostVitalByID(ctx context.Context, id uint) (*fleet.CustomHostVital, error) {
	vitals, err := svc.ds.GetCustomHostVitals(ctx, []uint{id})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get custom host vital by id")
	}
	if len(vitals) == 0 {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("CustomHostVital").WithID(id))
	}
	return &vitals[0], nil
}
