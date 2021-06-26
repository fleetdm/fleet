package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, spec := range specs {
		if spec.LabelMembershipType == fleet.LabelMembershipTypeDynamic && len(spec.Hosts) > 0 {
			return errors.Errorf("label %s is declared as dynamic but contains `hosts` key", spec.Name)
		}
		if spec.LabelMembershipType == fleet.LabelMembershipTypeManual && spec.Hosts == nil {
			// Hosts list doesn't need to contain anything, but it should at least not be nil.
			return errors.Errorf("label %s is declared as manual but contains no `hosts key`", spec.Name)
		}
	}
	return svc.ds.ApplyLabelSpecs(specs)
}

func (svc *Service) GetLabelSpecs(ctx context.Context) ([]*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpecs()
}

func (svc *Service) GetLabelSpec(ctx context.Context, name string) (*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpec(name)
}

func (svc *Service) NewLabel(ctx context.Context, p fleet.LabelPayload) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	label := &fleet.Label{}

	if p.Name == nil {
		return nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	label.Name = *p.Name

	if p.Query == nil {
		return nil, fleet.NewInvalidArgumentError("query", "missing required argument")
	}
	label.Query = *p.Query

	if p.Platform != nil {
		label.Platform = *p.Platform
	}

	if p.Description != nil {
		label.Description = *p.Description
	}

	label, err := svc.ds.NewLabel(label)
	if err != nil {
		return nil, err
	}
	return label, nil
}

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload fleet.ModifyLabelPayload) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	label, err := svc.ds.Label(id)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		label.Name = *payload.Name
	}
	if payload.Description != nil {
		label.Description = *payload.Description
	}
	return svc.ds.SaveLabel(label)
}

func (svc *Service) ListLabels(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.ListLabels(filter, opt)
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Label(id)
}

func (svc *Service) DeleteLabel(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteLabel(name)
}

func (svc *Service) DeleteLabelByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	label, err := svc.ds.Label(id)
	if err != nil {
		return err
	}
	return svc.ds.DeleteLabel(label.Name)
}

func (svc *Service) ListHostsInLabel(ctx context.Context, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.ListHostsInLabel(filter, lid, opt)
}

func (svc *Service) ListLabelsForHost(ctx context.Context, hid uint) ([]*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListLabelsForHost(hid)
}
