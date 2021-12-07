package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get Pack
////////////////////////////////////////////////////////////////////////////////

type getPackRequest struct {
	ID uint `url:"id"`
}

type getPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r getPackResponse) error() error { return r.Err }

func getPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getPackRequest)
	pack, err := svc.GetPack(ctx, req.ID)
	if err != nil {
		return getPackResponse{Err: err}, nil
	}

	resp, err := packResponseForPack(ctx, svc, *pack)
	if err != nil {
		return getPackResponse{Err: err}, nil
	}

	return getPackResponse{
		Pack: *resp,
	}, nil
}

func (svc *Service) GetPack(ctx context.Context, id uint) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Pack(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Create Pack
////////////////////////////////////////////////////////////////////////////////

type createPackRequest struct {
	fleet.PackPayload
}

type createPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r createPackResponse) error() error { return r.Err }

func createPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createPackRequest)
	pack, err := svc.NewPack(ctx, req.PackPayload)
	if err != nil {
		return createPackResponse{Err: err}, nil
	}

	resp, err := packResponseForPack(ctx, svc, *pack)
	if err != nil {
		return createPackResponse{Err: err}, nil
	}

	return createPackResponse{
		Pack: *resp,
	}, nil
}

func (svc *Service) NewPack(ctx context.Context, p fleet.PackPayload) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	var pack fleet.Pack

	if p.Name != nil {
		pack.Name = *p.Name
	}

	if p.Description != nil {
		pack.Description = *p.Description
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	if p.Disabled != nil {
		pack.Disabled = *p.Disabled
	}

	if p.HostIDs != nil {
		pack.HostIDs = *p.HostIDs
	}

	if p.LabelIDs != nil {
		pack.LabelIDs = *p.LabelIDs
	}

	if p.TeamIDs != nil {
		pack.TeamIDs = *p.TeamIDs
	}

	_, err := svc.ds.NewPack(ctx, &pack)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedPack,
		&map[string]interface{}{"pack_id": pack.ID, "pack_name": pack.Name},
	); err != nil {
		return nil, err
	}

	return &pack, nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify Pack
////////////////////////////////////////////////////////////////////////////////

type modifyPackRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.PackPayload
}

type modifyPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r modifyPackResponse) error() error { return r.Err }

func modifyPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyPackRequest)
	pack, err := svc.ModifyPack(ctx, req.ID, req.PackPayload)
	if err != nil {
		return modifyPackResponse{Err: err}, nil
	}

	resp, err := packResponseForPack(ctx, svc, *pack)
	if err != nil {
		return modifyPackResponse{Err: err}, nil
	}

	return modifyPackResponse{
		Pack: *resp,
	}, nil
}

func (svc *Service) ModifyPack(ctx context.Context, id uint, p fleet.PackPayload) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	pack, err := svc.ds.Pack(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil && pack.EditablePackType() {
		pack.Name = *p.Name
	}

	if p.Description != nil && pack.EditablePackType() {
		pack.Description = *p.Description
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	if p.Disabled != nil {
		pack.Disabled = *p.Disabled
	}

	if p.HostIDs != nil && pack.EditablePackType() {
		pack.HostIDs = *p.HostIDs
	}

	if p.LabelIDs != nil && pack.EditablePackType() {
		pack.LabelIDs = *p.LabelIDs
	}

	if p.TeamIDs != nil && pack.EditablePackType() {
		pack.TeamIDs = *p.TeamIDs
	}

	err = svc.ds.SavePack(ctx, pack)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedPack,
		&map[string]interface{}{"pack_id": pack.ID, "pack_name": pack.Name},
	); err != nil {
		return nil, err
	}

	return pack, err
}

////////////////////////////////////////////////////////////////////////////////
// List Packs
////////////////////////////////////////////////////////////////////////////////

type listPacksRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listPacksResponse struct {
	Packs []packResponse `json:"packs"`
	Err   error          `json:"error,omitempty"`
}

func (r listPacksResponse) error() error { return r.Err }

func listPacksEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listPacksRequest)
	packs, err := svc.ListPacks(ctx, fleet.PackListOptions{ListOptions: req.ListOptions, IncludeSystemPacks: false})
	if err != nil {
		return getPackResponse{Err: err}, nil
	}

	resp := listPacksResponse{Packs: make([]packResponse, len(packs))}
	for i, pack := range packs {
		packResp, err := packResponseForPack(ctx, svc, *pack)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}
		resp.Packs[i] = *packResp
	}
	return resp, nil
}

func (svc *Service) ListPacks(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacks(ctx, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Delete Pack
////////////////////////////////////////////////////////////////////////////////

type deletePackRequest struct {
	Name string `url:"name"`
}

type deletePackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deletePackResponse) error() error { return r.Err }

func deletePackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deletePackRequest)
	err := svc.DeletePack(ctx, req.Name)
	if err != nil {
		return deletePackResponse{Err: err}, nil
	}
	return deletePackResponse{}, nil
}

func (svc *Service) DeletePack(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	pack, _, err := svc.ds.PackByName(ctx, name)
	if err != nil {
		return err
	}
	// if there is a pack by this name, ensure it is not type Global or Team
	if pack != nil && !pack.EditablePackType() {
		return fmt.Errorf("cannot delete pack_type %s", *pack.Type)
	}

	if err := svc.ds.DeletePack(ctx, name); err != nil {
		return err
	}

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedPack,
		&map[string]interface{}{"pack_name": name},
	)
}

////////////////////////////////////////////////////////////////////////////////
// Delete Pack By ID
////////////////////////////////////////////////////////////////////////////////

type deletePackByIDRequest struct {
	ID uint `url:"id"`
}

type deletePackByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deletePackByIDResponse) error() error { return r.Err }

func deletePackByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deletePackByIDRequest)
	err := svc.DeletePackByID(ctx, req.ID)
	if err != nil {
		return deletePackByIDResponse{Err: err}, nil
	}
	return deletePackByIDResponse{}, nil
}

func (svc *Service) DeletePackByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	pack, err := svc.ds.Pack(ctx, id)
	if err != nil {
		return err
	}
	if pack != nil && !pack.EditablePackType() {
		return fmt.Errorf("cannot delete pack_type %s", *pack.Type)
	}
	if err := svc.ds.DeletePack(ctx, pack.Name); err != nil {
		return err
	}

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedPack,
		&map[string]interface{}{"pack_name": pack.Name},
	)
}

////////////////////////////////////////////////////////////////////////////////
// Apply Pack Spec
////////////////////////////////////////////////////////////////////////////////

type applyPackSpecsRequest struct {
	Specs []*fleet.PackSpec `json:"specs"`
}

type applyPackSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyPackSpecsResponse) error() error { return r.Err }

func applyPackSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyPackSpecsRequest)
	_, err := svc.ApplyPackSpecs(ctx, req.Specs)
	if err != nil {
		return applyPackSpecsResponse{Err: err}, nil
	}
	return applyPackSpecsResponse{}, nil
}

func (svc *Service) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) ([]*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	packs, err := svc.ds.ListPacks(ctx, fleet.PackListOptions{IncludeSystemPacks: true})
	if err != nil {
		return nil, err
	}

	namePacks := make(map[string]*fleet.Pack, len(packs))
	for _, pack := range packs {
		namePacks[pack.Name] = pack
	}

	var result []*fleet.PackSpec

	// loop over incoming specs filtering out possible edits to Global or Team Packs
	for _, spec := range specs {
		// see for known limitations https://github.com/fleetdm/fleet/pull/1558#discussion_r684218301
		// check to see if incoming spec is already in the list of packs
		if p, ok := namePacks[spec.Name]; ok {
			// as long as pack is editable, we'll apply it
			if p.EditablePackType() {
				result = append(result, spec)
			}
		} else {
			// incoming spec is new, let's apply it
			result = append(result, spec)
		}
	}

	if err := svc.ds.ApplyPackSpecs(ctx, result); err != nil {
		return nil, err
	}

	return result, svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAppliedSpecPack,
		&map[string]interface{}{},
	)
}

////////////////////////////////////////////////////////////////////////////////
// Get Pack Specs
////////////////////////////////////////////////////////////////////////////////

type getPackSpecsResponse struct {
	Specs []*fleet.PackSpec `json:"specs"`
	Err   error             `json:"error,omitempty"`
}

func (r getPackSpecsResponse) error() error { return r.Err }

func getPackSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	specs, err := svc.GetPackSpecs(ctx)
	if err != nil {
		return getPackSpecsResponse{Err: err}, nil
	}
	return getPackSpecsResponse{Specs: specs}, nil
}

func (svc *Service) GetPackSpecs(ctx context.Context) ([]*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpecs(ctx)
}
