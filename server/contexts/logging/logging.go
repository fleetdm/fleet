package logging

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
)

type key int

const loggingKey key = 0

// NewContext creates a new context.Context with a LoggingContext.
func NewContext(ctx context.Context, logging *LoggingContext) context.Context {
	return context.WithValue(ctx, loggingKey, logging)
}

// FromContext returns a pointer to the LoggingContext.
func FromContext(ctx context.Context) (*LoggingContext, bool) {
	v, ok := ctx.Value(loggingKey).(*LoggingContext)
	return v, ok
}

// WithStartTime returns a context with logging.StartTime marked as the current time
func WithStartTime(ctx context.Context) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.StartTime = time.Now()
	}
	return ctx
}

// WithErr returns a context with logging.Err set as the error provided
func WithErr(ctx context.Context, err ...error) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.Errs = append(logCtx.Errs, err...)
	}
	return ctx
}

// WithNoUser returns a context with logging.SkipUser set to true so user won't be logged
func WithNoUser(ctx context.Context) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.SkipUser = true
	}
	return ctx
}

// WithExtras returns a context with logging.Extras set as the values provided
func WithExtras(ctx context.Context, extras ...interface{}) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.Extras = append(logCtx.Extras, extras...)
	}
	return ctx
}

func WithLevel(ctx context.Context, level func(kitlog.Logger) kitlog.Logger) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.ForceLevel = level
	}
	return ctx
}

// LoggingContext contains the context information for logging the current request
type LoggingContext struct {
	StartTime  time.Time
	Errs       []error
	Extras     []interface{}
	SkipUser   bool
	ForceLevel func(kitlog.Logger) kitlog.Logger
}

// Log logs the data within the context
func (l *LoggingContext) Log(ctx context.Context, logger kitlog.Logger) {
	if l.ForceLevel != nil {
		logger = l.ForceLevel(logger)
	} else if l.Errs != nil || len(l.Extras) > 0 {
		logger = level.Info(logger)
	} else {
		logger = level.Debug(logger)
	}

	var keyvals []interface{}

	if !l.SkipUser {
		loggedInUser := "unauthenticated"
		vc, ok := viewer.FromContext(ctx)
		if ok {
			loggedInUser = vc.Email()
		}
		keyvals = append(keyvals, "user", loggedInUser)
	}

	requestMethod, ok := ctx.Value(kithttp.ContextKeyRequestMethod).(string)
	if !ok {
		requestMethod = ""
	}
	requestURI, ok := ctx.Value(kithttp.ContextKeyRequestURI).(string)
	if !ok {
		requestURI = ""
	}
	keyvals = append(keyvals, "method", requestMethod, "uri", requestURI, "took", time.Since(l.StartTime))

	if len(l.Extras) > 0 {
		keyvals = append(keyvals, l.Extras...)
	}

	if len(l.Errs) > 0 {
		var errs []string
		var internalErrs []string
		for _, err := range l.Errs {
			if e, ok := err.(fleet.ErrWithInternal); ok {
				internalErrs = append(internalErrs, e.Internal())
			} else {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			keyvals = append(keyvals, "err", errs)
		}
		if len(internalErrs) > 0 {
			keyvals = append(keyvals, "internal", internalErrs)
		}
	}

	_ = logger.Log(keyvals...)
}
