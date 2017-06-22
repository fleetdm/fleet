package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) ListLabels(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Label, error) {
	return svc.ds.ListLabels(opt)
}

func (svc service) GetLabel(ctx context.Context, id uint) (*kolide.Label, error) {
	return svc.ds.Label(id)
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

func (svc service) DeleteLabel(ctx context.Context, id uint) error {
	return svc.ds.DeleteLabel(id)
}

func (svc service) HostIDsForLabel(lid uint) ([]uint, error) {
	hosts, err := svc.ds.ListHostsInLabel(lid)
	if err != nil {
		return nil, err
	}
	var ids []uint
	for _, h := range hosts {
		ids = append(ids, h.ID)
	}
	return ids, nil
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
