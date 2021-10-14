package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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

func (svc *Service) GetPackSpecs(ctx context.Context) ([]*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpecs(ctx)
}

func (svc *Service) GetPackSpec(ctx context.Context, name string) (*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpec(ctx, name)
}

func (svc *Service) ListPacks(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacks(ctx, opt)
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

func (svc *Service) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacksForHost(ctx, hid)
}
