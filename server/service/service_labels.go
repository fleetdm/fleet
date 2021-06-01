package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc *Service) ApplyLabelSpecs(ctx context.Context, specs []*kolide.LabelSpec) error {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "write"); err != nil {
		return err
	}

	for _, spec := range specs {
		if spec.LabelMembershipType == kolide.LabelMembershipTypeDynamic && len(spec.Hosts) > 0 {
			return errors.Errorf("label %s is declared as dynamic but contains `hosts` key", spec.Name)
		}
		if spec.LabelMembershipType == kolide.LabelMembershipTypeManual && spec.Hosts == nil {
			// Hosts list doesn't need to contain anything, but it should at least not be nil.
			return errors.Errorf("label %s is declared as manual but contains not `hosts key`", spec.Name)
		}
	}
	return svc.ds.ApplyLabelSpecs(specs)
}

func (svc *Service) GetLabelSpecs(ctx context.Context) ([]*kolide.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpecs()
}

func (svc *Service) GetLabelSpec(ctx context.Context, name string) (*kolide.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpec(name)
}

func (svc *Service) NewLabel(ctx context.Context, p kolide.LabelPayload) (*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "write"); err != nil {
		return nil, err
	}

	label := &kolide.Label{}

	if p.Name == nil {
		return nil, kolide.NewInvalidArgumentError("name", "missing required argument")
	}
	label.Name = *p.Name

	if p.Query == nil {
		return nil, kolide.NewInvalidArgumentError("query", "missing required argument")
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

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload kolide.ModifyLabelPayload) (*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "write"); err != nil {
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

func (svc *Service) ListLabels(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Team{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListLabels(opt)
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Team{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.Label(id)
}

func (svc *Service) DeleteLabel(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "write"); err != nil {
		return err
	}

	return svc.ds.DeleteLabel(name)
}

func (svc *Service) DeleteLabelByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Label{}, "write"); err != nil {
		return err
	}

	label, err := svc.ds.Label(id)
	if err != nil {
		return err
	}
	return svc.ds.DeleteLabel(label.Name)
}

func (svc *Service) ListHostsInLabel(ctx context.Context, lid uint, opt kolide.HostListOptions) ([]*kolide.Host, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Team{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListHostsInLabel(lid, opt)
}

func (svc *Service) ListLabelsForHost(ctx context.Context, hid uint) ([]*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Team{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListLabelsForHost(hid)
}
