package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.ApplyPackSpecs(specs)
}

func (svc *Service) GetPackSpecs(ctx context.Context) ([]*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpecs()
}

func (svc *Service) GetPackSpec(ctx context.Context, name string) (*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpec(name)
}

func (svc *Service) ListPacks(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacks(opt)
}

func (svc *Service) GetPack(ctx context.Context, id uint) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Pack(id)
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

	_, err := svc.ds.NewPack(&pack)
	if err != nil {
		return nil, err
	}

	return &pack, nil
}

func (svc *Service) ModifyPack(ctx context.Context, id uint, p fleet.PackPayload) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	pack, err := svc.ds.Pack(id)
	if err != nil {
		return nil, err
	}

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

	err = svc.ds.SavePack(pack)
	if err != nil {
		return nil, err
	}

	return pack, err
}

func (svc *Service) DeletePack(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeletePack(name)
}

func (svc *Service) DeletePackByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	pack, err := svc.ds.Pack(id)
	if err != nil {
		return err
	}
	return svc.ds.DeletePack(pack.Name)
}

func (svc *Service) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacksForHost(hid)
}
