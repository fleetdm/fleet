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
