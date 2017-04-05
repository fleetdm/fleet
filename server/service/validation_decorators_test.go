package service

import (
	"context"
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dtPtr = func(t kolide.DecoratorType) *kolide.DecoratorType { return &t }

func TestDecoratorValidation(t *testing.T) {
	ds := mock.Store{}
	ds.DecoratorFunc = func(id uint) (*kolide.Decorator, error) {
		return &kolide.Decorator{
			ID:    1,
			Query: "select x from y;",
			Type:  kolide.DecoratorAlways,
		}, nil
	}
	ds.SaveDecoratorFunc = func(dec *kolide.Decorator) error {
		return nil
	}
	svc := &service{
		ds: &ds,
	}
	validator := validationMiddleware{
		Service: svc,
		ds:      &ds,
	}

	payload := kolide.DecoratorPayload{
		ID:            uint(1),
		DecoratorType: dtPtr(kolide.DecoratorInterval),
		Interval:      uintPtr(3600),
	}

	dec, err := validator.ModifyDecorator(context.Background(), payload)
	require.Nil(t, err)
	assert.Equal(t, kolide.DecoratorInterval, dec.Type)
	assert.Equal(t, uint(3600), dec.Interval)
}

func TestDecoratorValidationIntervalMissing(t *testing.T) {
	ds := mock.Store{}
	ds.DecoratorFunc = func(id uint) (*kolide.Decorator, error) {
		return &kolide.Decorator{
			ID:    1,
			Query: "select x from y;",
			Type:  kolide.DecoratorAlways,
		}, nil
	}
	ds.SaveDecoratorFunc = func(dec *kolide.Decorator) error {
		return nil
	}
	svc := &service{
		ds: &ds,
	}
	validator := validationMiddleware{
		Service: svc,
		ds:      &ds,
	}

	payload := kolide.DecoratorPayload{
		ID:            uint(1),
		DecoratorType: dtPtr(kolide.DecoratorInterval),
	}

	_, err := validator.ModifyDecorator(context.Background(), payload)
	require.NotNil(t, err)
	r, ok := err.(*invalidArgumentError)
	require.True(t, ok)
	assert.Equal(t, "missing required argument", (*r)[0].reason)
}

func TestDecoratorValidationIntervalSameType(t *testing.T) {
	ds := mock.Store{}
	ds.DecoratorFunc = func(id uint) (*kolide.Decorator, error) {
		return &kolide.Decorator{
			ID:       1,
			Query:    "select x from y;",
			Type:     kolide.DecoratorInterval,
			Interval: 600,
		}, nil
	}
	ds.SaveDecoratorFunc = func(dec *kolide.Decorator) error {
		return nil
	}
	svc := &service{
		ds: &ds,
	}
	validator := validationMiddleware{
		Service: svc,
		ds:      &ds,
	}

	payload := kolide.DecoratorPayload{
		ID:            uint(1),
		DecoratorType: dtPtr(kolide.DecoratorInterval),
		Interval:      uintPtr(1200),
	}

	dec, err := validator.ModifyDecorator(context.Background(), payload)
	require.Nil(t, err)
	assert.Equal(t, uint(1200), dec.Interval)
}

func TestDecoratorValidationIntervalInvalid(t *testing.T) {
	ds := mock.Store{}
	ds.DecoratorFunc = func(id uint) (*kolide.Decorator, error) {
		return &kolide.Decorator{
			ID:       1,
			Query:    "select x from y;",
			Type:     kolide.DecoratorInterval,
			Interval: 600,
		}, nil
	}
	ds.SaveDecoratorFunc = func(dec *kolide.Decorator) error {
		return nil
	}
	svc := &service{
		ds: &ds,
	}
	validator := validationMiddleware{
		Service: svc,
		ds:      &ds,
	}

	payload := kolide.DecoratorPayload{
		ID:            uint(1),
		DecoratorType: dtPtr(kolide.DecoratorInterval),
		Interval:      uintPtr(1203),
	}

	_, err := validator.ModifyDecorator(context.Background(), payload)
	require.NotNil(t, err)
	r, ok := err.(*invalidArgumentError)
	require.True(t, ok)
	assert.Equal(t, "value must be divisible by 60", (*r)[0].reason)
}
