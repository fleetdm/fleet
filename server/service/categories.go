package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

//////////////////////////////////////////////////////////////////////////////
// List self-service categories
//////////////////////////////////////////////////////////////////////////////

type getSelfServiceCategoriesRequest struct {
	FleetID *uint `query:"fleet_id"`
}

type getSelfServiceCategoriesResponse struct {
	SelfServiceCategories []*fleet.SoftwareCategory `json:"self_service_categories"`
	Err                   error                     `json:"error,omitempty"`
}

func (r getSelfServiceCategoriesResponse) Error() error { return r.Err }

func getSelfServiceCategoriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getSelfServiceCategoriesRequest)
	categories, err := svc.ListSoftwareCategories(ctx, ptr.ValOrZero(req.FleetID))
	if err != nil {
		return getSelfServiceCategoriesResponse{Err: err}, nil
	}
	return getSelfServiceCategoriesResponse{SelfServiceCategories: categories}, nil
}

func (svc *Service) ListSoftwareCategories(ctx context.Context, _ uint) ([]*fleet.SoftwareCategory, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Add self-service category
//////////////////////////////////////////////////////////////////////////////

type addSelfServiceCategoriesRequest struct {
	FleetID *uint  `json:"fleet_id"`
	Name    string `json:"name"`
}

type addSelfServiceCategoriesResponse struct {
	SelfServiceCategory *fleet.SoftwareCategory `json:"self_service_category"`
	Err                 error                   `json:"error,omitempty"`
}

func (r addSelfServiceCategoriesResponse) Error() error { return r.Err }

func addSelfServiceCategoriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*addSelfServiceCategoriesRequest)
	category, err := svc.NewSoftwareCategory(ctx, ptr.ValOrZero(req.FleetID), req.Name)
	if err != nil {
		return addSelfServiceCategoriesResponse{Err: err}, nil
	}
	return addSelfServiceCategoriesResponse{SelfServiceCategory: category}, nil
}

func (svc *Service) NewSoftwareCategory(ctx context.Context, _ uint, _ string) (*fleet.SoftwareCategory, error) {
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

func patchSelfServiceCategoriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func deleteSelfServiceCategoriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
