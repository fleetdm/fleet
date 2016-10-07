package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) ListLabels(ctx context.Context) ([]*kolide.Label, error) {
	return svc.ds.Labels()
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

	if p.QueryID != nil {
		label.QueryID = *p.QueryID
	}

	label, err := svc.ds.NewLabel(label)
	if err != nil {
		return nil, err
	}
	return label, nil
}

func (svc service) ModifyLabel(ctx context.Context, id uint, p kolide.LabelPayload) (*kolide.Label, error) {
	label, err := svc.ds.Label(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		label.Name = *p.Name
	}

	if p.QueryID != nil {
		label.QueryID = *p.QueryID
	}

	err = svc.ds.SaveLabel(label)
	if err != nil {
		return nil, err
	}

	return label, nil
}

func (svc service) DeleteLabel(ctx context.Context, id uint) error {
	return svc.ds.DeleteLabel(id)
}
