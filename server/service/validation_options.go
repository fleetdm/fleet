package service

import (
	"context"
	"errors"

	"github.com/kolide/kolide/server/kolide"
)

func (mw validationMiddleware) ModifyOptions(ctx context.Context, req kolide.OptionRequest) ([]kolide.Option, error) {
	invalid := &invalidArgumentError{}
	for _, opt := range req.Options {
		if opt.ReadOnly {
			invalid.Append(opt.Name, "readonly option")
			continue
		}
		if err := validateValueMapsToOptionType(opt); err != nil {
			invalid.Append(opt.Name, err.Error())
		}
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyOptions(ctx, req)
}

var (
	errTypeMismatch = errors.New("type mismatch")
	errInvalidType  = errors.New("invalid option type")
)

func validateValueMapsToOptionType(opt kolide.Option) error {
	if !opt.OptionSet() {
		return nil
	}
	switch opt.GetValue().(type) {
	case int, uint, uint64, float64:
		if opt.Type != kolide.OptionTypeInt {
			return errTypeMismatch
		}
	case string:
		if opt.Type != kolide.OptionTypeString {
			return errTypeMismatch
		}
	case bool:
		if opt.Type != kolide.OptionTypeBool {
			return errTypeMismatch
		}
	default:
		return errInvalidType
	}
	return nil
}
