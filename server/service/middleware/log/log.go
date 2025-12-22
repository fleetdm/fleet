package log

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
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

// NewLoggingMiddleware creates middleware that initializes logging context and logs requests.
// This middleware should be used for raw http.Handler endpoints (not go-kit endpoints).
// It initializes the logging context with request metadata and logs the request after completion.
func NewLoggingMiddleware(logger kitlog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create logging context
			lc := &logging.LoggingContext{}

			// Populate request context with kithttp fields (method, URI, etc.)
			ctx := logging.NewContext(kithttp.PopulateRequestContext(r.Context(), r), lc)

			// Set start time
			ctx = logging.WithStartTime(ctx)

			// Add IP address information to logging context
			remoteAddr, _ := ctx.Value(kithttp.ContextKeyRequestRemoteAddr).(string)
			xForwardedFor, _ := ctx.Value(kithttp.ContextKeyRequestXForwardedFor).(string)
			lc.SetExtras("ip_addr", remoteAddr, "x_for_ip_addr", xForwardedFor)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)

			// Log the request
			LogRequestEnd(logger)(ctx, w)
		})
	}
}
