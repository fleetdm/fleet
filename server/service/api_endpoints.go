package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//////////////////////////////////////////////////////////////////////////////////
// List API endpoints
//////////////////////////////////////////////////////////////////////////////////

type listAPIEndpointsRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listAPIEndpointsResponse struct {
	ApiEndpoints []fleet.APIEndpoint       `json:"api_endpoints"`
	Meta         *fleet.PaginationMetadata `json:"meta"`
	Count        int                       `json:"count"`
	Err          error                     `json:"error,omitempty"`
}

func (r listAPIEndpointsResponse) Error() error { return r.Err }

func listAPIEndpointsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listAPIEndpointsRequest)
	endpoints, meta, count, err := svc.ListAPIEndpoints(ctx, req.ListOptions)
	return listAPIEndpointsResponse{
		ApiEndpoints: endpoints,
		Meta:         meta,
		Count:        count,
		Err:          err,
	}, nil
}

func (svc *Service) ListAPIEndpoints(ctx context.Context, opts fleet.ListOptions) ([]fleet.APIEndpoint, *fleet.PaginationMetadata, int, error) {
	// skipauth: No authorization check, this is a premium feature only
	svc.authz.SkipAuthorization(ctx)
	return nil, nil, 0, fleet.ErrMissingLicense
}
