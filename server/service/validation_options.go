package service

import (
	"fmt"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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
	errTypeMismatch = fmt.Errorf("type mismatch")
	errInvalidType  = fmt.Errorf("invalid option type")
)

func validateValueMapsToOptionType(opt kolide.Option) error {
	if !opt.OptionSet() {
		return nil
	}
	val := opt.GetValue()
	switch opt.Type {
	case kolide.OptionTypeBool:
		_, ok := val.(bool)
		if !ok {
			return errTypeMismatch
		}
	case kolide.OptionTypeString:
		_, ok := val.(string)
		if !ok {
			return errTypeMismatch
		}
	case kolide.OptionTypeInt:
		_, ok := val.(float64) // JSON unmarshaler represents all numbers in float64
		if !ok {
			return errTypeMismatch
		}
	default:
		return errInvalidType
	}
	return nil
}
