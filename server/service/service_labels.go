package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ApplyLabelSpecs(ctx context.Context, specs []*kolide.LabelSpec) error {
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

func (svc service) GetLabelSpecs(ctx context.Context) ([]*kolide.LabelSpec, error) {
	return svc.ds.GetLabelSpecs()
}

func (svc service) GetLabelSpec(ctx context.Context, name string) (*kolide.LabelSpec, error) {
	return svc.ds.GetLabelSpec(name)
}

func (svc service) NewLabel(ctx context.Context, p kolide.LabelPayload) (*kolide.Label, error) {
	label := &kolide.Label{}

	if p.Name == nil {
		return nil, newInvalidArgumentError("name", "missing required argument")
	}
	label.Name = *p.Name

	if p.Query == nil {
		return nil, newInvalidArgumentError("query", "missing required argument")
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

func (svc service) ModifyLabel(ctx context.Context, id uint, payload kolide.ModifyLabelPayload) (*kolide.Label, error) {
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

func (svc service) ListLabels(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Label, error) {
	return svc.ds.ListLabels(opt)
}

func (svc service) GetLabel(ctx context.Context, id uint) (*kolide.Label, error) {
	return svc.ds.Label(id)
}

func (svc service) DeleteLabel(ctx context.Context, name string) error {
	return svc.ds.DeleteLabel(name)
}

func (svc service) DeleteLabelByID(ctx context.Context, id uint) error {
	label, err := svc.ds.Label(id)
	if err != nil {
		return err
	}
	return svc.ds.DeleteLabel(label.Name)
}

func (svc service) ListHostsInLabel(ctx context.Context, lid uint, opt kolide.HostListOptions) ([]kolide.Host, error) {
	return svc.ds.ListHostsInLabel(lid, opt)
}

func (svc service) ListLabelsForHost(ctx context.Context, hid uint) ([]kolide.Label, error) {
	return svc.ds.ListLabelsForHost(hid)
}

func (svc service) HostIDsForLabel(lid uint) ([]uint, error) {
	hosts, err := svc.ds.ListHostsInLabel(lid, kolide.HostListOptions{})
	if err != nil {
		return nil, err
	}
	var ids []uint
	for _, h := range hosts {
		ids = append(ids, h.ID)
	}
	return ids, nil
}
