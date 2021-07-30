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
func WithErr(ctx context.Context, err error) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.Err = err
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

// WithExtrasNoUser returns a context with logging.Extras set as the values
// provided and skips the user logging
func WithExtrasNoUser(ctx context.Context, extras ...interface{}) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.Extras = append(logCtx.Extras, extras...)
		logCtx.SkipUser = true
	}
	return ctx
}

// WithDebugExtrasNoUser returns a context with logging.Extras set as the values
// provided, skips the user logging, and forces the log to be a debug
func WithDebugExtrasNoUser(ctx context.Context, extras ...interface{}) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.Extras = append(logCtx.Extras, extras...)
		logCtx.ForceDebug = true
		logCtx.SkipUser = true
	}
	return ctx
}

// LoggingContext contains the context information for logging the current request
type LoggingContext struct {
	StartTime  time.Time
	Err        error
	Extras     []interface{}
	ForceDebug bool
	SkipUser   bool
}

// Log logs the data within the context
func (l *LoggingContext) Log(ctx context.Context, logger kitlog.Logger) {
	if e, ok := l.Err.(fleet.ErrWithInternal); ok {
		logger = kitlog.With(logger, "internal", e.Internal())
	}

	if (l.Err != nil || len(l.Extras) > 0) && !l.ForceDebug {
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

	keyvals = append(keyvals,
		"method", ctx.Value(kithttp.ContextKeyRequestMethod).(string),
		"uri", ctx.Value(kithttp.ContextKeyRequestURI).(string),
		"took", time.Since(l.StartTime),
	)

	if l.Err != nil {
		keyvals = append(keyvals, "err", l.Err)
	}

	if len(l.Extras) > 0 {
		keyvals = append(keyvals, l.Extras...)
	}

	_ = logger.Log(keyvals...)
}
