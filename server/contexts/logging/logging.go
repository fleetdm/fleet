package logging

import (
	"context"
	"errors"
	"strings"
	"sync"
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
		logCtx.SetStartTime()
	}
	return ctx
}

// WithErr returns a context with logging.Err set as the error provided
func WithErr(ctx context.Context, err ...error) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.SetErrs(err...)
	}
	return ctx
}

// WithNoUser returns a context with logging.SkipUser set to true so user won't be logged
func WithNoUser(ctx context.Context) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.SetSkipUser()
	}
	return ctx
}

// WithExtras returns a context with logging.Extras set as the values provided
func WithExtras(ctx context.Context, extras ...interface{}) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.SetExtras(extras...)
	}
	return ctx
}

func WithLevel(ctx context.Context, level func(kitlog.Logger) kitlog.Logger) context.Context {
	if logCtx, ok := FromContext(ctx); ok {
		logCtx.SetForceLevel(level)
	}
	return ctx
}

// LoggingContext contains the context information for logging the current request
type LoggingContext struct {
	l sync.Mutex

	StartTime  time.Time
	Errs       []error
	Extras     []interface{}
	SkipUser   bool
	ForceLevel func(kitlog.Logger) kitlog.Logger
}

func (l *LoggingContext) SetForceLevel(level func(kitlog.Logger) kitlog.Logger) {
	l.l.Lock()
	defer l.l.Unlock()
	l.ForceLevel = level
}

func (l *LoggingContext) SetExtras(extras ...interface{}) {
	l.l.Lock()
	defer l.l.Unlock()
	l.Extras = append(l.Extras, extras...)
}

func (l *LoggingContext) SetSkipUser() {
	l.l.Lock()
	defer l.l.Unlock()
	l.SkipUser = true
}

func (l *LoggingContext) SetStartTime() {
	l.l.Lock()
	defer l.l.Unlock()
	l.StartTime = time.Now()
}

func (l *LoggingContext) SetErrs(err ...error) {
	l.l.Lock()
	defer l.l.Unlock()
	l.Errs = append(l.Errs, err...)
}

// Log logs the data within the context
func (l *LoggingContext) Log(ctx context.Context, logger kitlog.Logger) {
	l.l.Lock()
	defer l.l.Unlock()

	switch {
	case len(l.Errs) > 0:
		logger = level.Error(logger)
	case l.ForceLevel != nil:
		logger = l.ForceLevel(logger)
	default:
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
		// Going for string concatenation here instead of json.Marshal mostly to not have to deal with error handling
		// within this method. kitlog doesn't support slices of strings
		var (
			errs         string
			internalErrs string
			uuids        []string
		)
		separator := " || "
		for _, err := range l.Errs {
			var ewi fleet.ErrWithInternal
			if errors.As(err, &ewi) {
				if internalErrs == "" {
					internalErrs = ewi.Internal()
				} else {
					internalErrs += separator + ewi.Internal()
				}
			} else {
				if errs == "" {
					errs = err.Error()
				} else {
					errs += separator + err.Error()
				}
			}
			var ewuuid fleet.ErrorUUIDer
			if errors.As(err, &ewuuid) {
				if uuid := ewuuid.UUID(); uuid != "" {
					uuids = append(uuids, uuid)
				}
			}
		}
		if len(errs) > 0 {
			keyvals = append(keyvals, "err", errs)
		}
		if len(internalErrs) > 0 {
			keyvals = append(keyvals, "internal", internalErrs)
		}
		if len(uuids) > 0 {
			keyvals = append(keyvals, "uuid", strings.Join(uuids, ","))
		}
	}

	_ = logger.Log(keyvals...)
}
