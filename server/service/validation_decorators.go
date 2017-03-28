package service

import (
	"github.com/kolide/kolide/server/kolide"
	"golang.org/x/net/context"
)

func validateNewDecoratorType(payload kolide.DecoratorPayload, invalid *invalidArgumentError) {
	if payload.DecoratorType == nil {
		invalid.Append("decorator_type", "missing required argument")
		return
	}
	if *payload.DecoratorType == kolide.DecoratorUndefined {
		invalid.Append("decorator_type", "invalid value, must be load, always, or interval")
		return
	}
	if *payload.DecoratorType == kolide.DecoratorInterval {
		if payload.Interval == nil {
			invalid.Append("interval", "missing required argument")
			return
		}
		if *payload.Interval%60 != 0 {
			invalid.Append("interval", "must be divisible by 60")
			return
		}
	}
}

// NewDecorator validator checks to make sure that a valid decorator type exists and
// if the decorator is of an interval type, an interval value is present and is
// divisable by 60
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
func (mw validationMiddleware) NewDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	invalid := &invalidArgumentError{}
	validateNewDecoratorType(payload, invalid)

	if payload.Query == nil {
		invalid.Append("query", "missing required argument")
	}

	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.NewDecorator(ctx, payload)
}

func (mw validationMiddleware) validateModifyDecoratorType(payload kolide.DecoratorPayload, invalid *invalidArgumentError) error {
	if payload.DecoratorType != nil {

		if *payload.DecoratorType == kolide.DecoratorUndefined {
			invalid.Append("decorator_type", "invalid value, must be load, always, or interval")
			return nil
		}
		if *payload.DecoratorType == kolide.DecoratorInterval {
			// special processing for interval type
			existingDec, err := mw.ds.Decorator(payload.ID)
			if err != nil {
				// if decorator is not present we want to return a 404 to the client
				return err
			}
			// if the type has changed from always or load to interval we need to
			// check suitability of interval value
			if existingDec.Type != kolide.DecoratorInterval {
				if payload.Interval == nil {
					invalid.Append("interval", "missing required argument")
					return nil
				}
			}
		}

		if payload.Interval != nil {
			if *payload.Interval%60 != 0 {
				invalid.Append("interval", "value must be divisible by 60")
			}
		}
	}
	return nil
}

func (mw validationMiddleware) ModifyDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	invalid := &invalidArgumentError{}
	err := mw.validateModifyDecoratorType(payload, invalid)
	if err != nil {
		return nil, err
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyDecorator(ctx, payload)
}

func (mw validationMiddleware) DeleteDecorator(ctx context.Context, id uint) error {
	invalid := &invalidArgumentError{}
	dec, err := mw.ds.Decorator(id)
	if err != nil {
		return err
	}
	if dec.BuiltIn {
		invalid.Append("decorator", "read only")
		return invalid
	}
	return mw.Service.DeleteDecorator(ctx, id)
}
