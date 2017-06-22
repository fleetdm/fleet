package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) ListDecorators(ctx context.Context) ([]*kolide.Decorator, error) {
	return svc.ds.ListDecorators()
}

func (svc service) DeleteDecorator(ctx context.Context, uid uint) error {
	return svc.ds.DeleteDecorator(uid)
}

func (svc service) NewDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	var dec kolide.Decorator
	if payload.Name != nil {
		dec.Name = *payload.Name
	}
	dec.Query = *payload.Query
	dec.Type = *payload.DecoratorType
	if payload.Interval != nil {
		dec.Interval = *payload.Interval
	}
	return svc.ds.NewDecorator(&dec)
}

func (svc service) ModifyDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	dec, err := svc.ds.Decorator(payload.ID)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		dec.Name = *payload.Name
	}
	if payload.DecoratorType != nil {
		dec.Type = *payload.DecoratorType
	}
	if payload.Query != nil {
		dec.Query = *payload.Query
	}
	if payload.Interval != nil {
		dec.Interval = *payload.Interval
	}
	err = svc.ds.SaveDecorator(dec)
	if err != nil {
		return nil, err
	}
	return dec, nil
}
