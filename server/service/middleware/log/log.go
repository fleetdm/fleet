package log

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/go-kit/kit/endpoint"
)

// Logged wraps an endpoint and adds the error if the context supports it
func Logged(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		res, err := next(ctx, request)
		if err != nil {
			logging.WithErr(ctx, err)
			return nil, err
		}
		if errResp, ok := res.(interface{ Error() error }); ok {
			err = errResp.Error()
			if err != nil {
				logging.WithErr(ctx, err)
			}
		}
		return res, nil
	}
}
