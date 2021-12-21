package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, spec := range specs {
		if spec.LabelMembershipType == fleet.LabelMembershipTypeDynamic && len(spec.Hosts) > 0 {
			return ctxerr.Errorf(ctx, "label %s is declared as dynamic but contains `hosts` key", spec.Name)
		}
		if spec.LabelMembershipType == fleet.LabelMembershipTypeManual && spec.Hosts == nil {
			// Hosts list doesn't need to contain anything, but it should at least not be nil.
			return ctxerr.Errorf(ctx, "label %s is declared as manual but contains no `hosts key`", spec.Name)
		}
	}
	return svc.ds.ApplyLabelSpecs(ctx, specs)
}

func (svc *Service) GetLabelSpecs(ctx context.Context) ([]*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpecs(ctx)
}

func (svc *Service) GetLabelSpec(ctx context.Context, name string) (*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpec(ctx, name)
}

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload fleet.ModifyLabelPayload) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	label, err := svc.ds.Label(ctx, id)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		label.Name = *payload.Name
	}
	if payload.Description != nil {
		label.Description = *payload.Description
	}
	return svc.ds.SaveLabel(ctx, label)
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

	return svc.ds.ListLabels(ctx, filter, opt)
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Label(ctx, id)
}

func (svc *Service) DeleteLabel(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteLabel(ctx, name)
}

func (svc *Service) DeleteLabelByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	label, err := svc.ds.Label(ctx, id)
	if err != nil {
		return err
	}
	return svc.ds.DeleteLabel(ctx, label.Name)
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

	return svc.ds.ListHostsInLabel(ctx, filter, lid, opt)
}

func (svc *Service) ListLabelsForHost(ctx context.Context, hid uint) ([]*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListLabelsForHost(ctx, hid)
}
