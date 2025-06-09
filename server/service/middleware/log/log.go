package log

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/log"
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

func LogRequestEnd(logger kitlog.Logger) func(context.Context, http.ResponseWriter) context.Context {
	return func(ctx context.Context, w http.ResponseWriter) context.Context {
		logCtx, ok := logging.FromContext(ctx)
		if !ok {
			return ctx
		}
		logCtx.Log(ctx, logger)
		return ctx
	}
}

func LogResponseEndMiddleware(logger kitlog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		LogRequestEnd(logger)(r.Context(), w)
	})
}
