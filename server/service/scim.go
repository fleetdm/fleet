package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func getScimDetailsEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	details, err := svc.ScimDetails(ctx)
	if err != nil {
		return fleet.ScimDetailsResponse{Err: err}, nil
	}
	return fleet.ScimDetailsResponse{
		ScimDetails: details,
	}, nil
}

func (svc *Service) ScimDetails(ctx context.Context) (fleet.ScimDetails, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ScimDetails{}, fleet.ErrMissingLicense
}
