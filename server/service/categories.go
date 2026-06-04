package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

//////////////////////////////////////////////////////////////////////////////
// List self-service categories
//////////////////////////////////////////////////////////////////////////////

type getSelfServiceCategoriesRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type getSelfServiceCategoriesResponse struct {
	SelfServiceCategories []fleet.SoftwareCategory `json:"self_service_categories"`
	Err                   error                    `json:"error,omitempty"`
}

func (r getSelfServiceCategoriesResponse) Error() error { return r.Err }

func getSelfServiceCategoriesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getSelfServiceCategoriesRequest)
	categories, err := svc.ListSoftwareCategories(ctx, req.TeamID)
	if err != nil {
		return getSelfServiceCategoriesResponse{Err: err}, nil
	}
	return getSelfServiceCategoriesResponse{SelfServiceCategories: categories}, nil
}

func (svc *Service) ListSoftwareCategories(ctx context.Context, _ *uint) ([]fleet.SoftwareCategory, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// List self-service categories for a device (token-authenticated)
//////////////////////////////////////////////////////////////////////////////

type getDeviceSelfServiceCategoriesRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceSelfServiceCategoriesRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceSelfServiceCategoriesResponse struct {
	SelfServiceCategories []fleet.SoftwareCategory `json:"self_service_categories"`
	Err                   error                    `json:"error,omitempty"`
}

func (r getDeviceSelfServiceCategoriesResponse) Error() error { return r.Err }

func getDeviceSelfServiceCategoriesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceSelfServiceCategoriesResponse{Err: err}, nil
	}

	categories, err := svc.ListSelfServiceSoftwareCategoriesForHost(ctx, host)
	if err != nil {
		return getDeviceSelfServiceCategoriesResponse{Err: err}, nil
	}
	return getDeviceSelfServiceCategoriesResponse{SelfServiceCategories: categories}, nil
}

func (svc *Service) ListSelfServiceSoftwareCategoriesForHost(ctx context.Context, host *fleet.Host) ([]fleet.SoftwareCategory, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Add self-service category
//////////////////////////////////////////////////////////////////////////////

type addSelfServiceCategoriesRequest struct {
	TeamID *uint  `json:"team_id" renameto:"fleet_id"`
	Name   string `json:"name"`
}

type addSelfServiceCategoriesResponse struct {
	SelfServiceCategory *fleet.SoftwareCategory `json:"self_service_category"`
	Err                 error                   `json:"error,omitempty"`
}

func (r addSelfServiceCategoriesResponse) Error() error { return r.Err }

func addSelfServiceCategoriesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*addSelfServiceCategoriesRequest)
	category, err := svc.NewSoftwareCategory(ctx, req.TeamID, req.Name)
	if err != nil {
		return addSelfServiceCategoriesResponse{Err: err}, nil
	}
	return addSelfServiceCategoriesResponse{SelfServiceCategory: category}, nil
}

func (svc *Service) NewSoftwareCategory(ctx context.Context, _ *uint, _ string) (*fleet.SoftwareCategory, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Update self-service category
//////////////////////////////////////////////////////////////////////////////

type patchSelfServiceCategoriesRequest struct {
	ID   uint   `url:"id"`
	Name string `json:"name"`
}

type patchSelfServiceCategoriesResponse struct {
	SelfServiceCategory *fleet.SoftwareCategory `json:"self_service_category"`
	Err                 error                   `json:"error,omitempty"`
}

func (r patchSelfServiceCategoriesResponse) Error() error { return r.Err }

func patchSelfServiceCategoriesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*patchSelfServiceCategoriesRequest)
	category, err := svc.UpdateSoftwareCategory(ctx, req.ID, req.Name)
	if err != nil {
		return patchSelfServiceCategoriesResponse{Err: err}, nil
	}
	return patchSelfServiceCategoriesResponse{SelfServiceCategory: category}, nil
}

func (svc *Service) UpdateSoftwareCategory(ctx context.Context, _ uint, _ string) (*fleet.SoftwareCategory, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Delete self-service category
//////////////////////////////////////////////////////////////////////////////

type deleteSelfServiceCategoriesRequest struct {
	ID uint `url:"id"`
}

type deleteSelfServiceCategoriesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSelfServiceCategoriesResponse) Error() error { return r.Err }
func (r deleteSelfServiceCategoriesResponse) Status() int  { return http.StatusNoContent }

func deleteSelfServiceCategoriesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteSelfServiceCategoriesRequest)
	if err := svc.DeleteSoftwareCategory(ctx, req.ID); err != nil {
		return deleteSelfServiceCategoriesResponse{Err: err}, nil
	}
	return deleteSelfServiceCategoriesResponse{}, nil
}

func (svc *Service) DeleteSoftwareCategory(ctx context.Context, _ uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
