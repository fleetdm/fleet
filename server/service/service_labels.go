package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) ApplyLabelSpecs(ctx context.Context, specs []*kolide.LabelSpec) error {
	return svc.ds.ApplyLabelSpecs(specs)
}

func (svc service) GetLabelSpecs(ctx context.Context) ([]*kolide.LabelSpec, error) {
	return svc.ds.GetLabelSpecs()
}

func (svc service) ListLabels(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Label, error) {
	return svc.ds.ListLabels(opt)
}

func (svc service) GetLabel(ctx context.Context, id uint) (*kolide.Label, error) {
	return svc.ds.Label(id)
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
