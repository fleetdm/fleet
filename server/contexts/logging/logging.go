package logging

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	kithttp "github.com/go-kit/kit/transport/http"
)

// UserEmailer provides the user's email for logging purposes.
type UserEmailer interface {
	Email() string
}

type userEmailerKey struct{}

// WithUserEmailer returns a context with the UserEmailer stored for logging.
// This should be called by authentication middleware after the user is identified.
func WithUserEmailer(ctx context.Context, emailer UserEmailer) context.Context {
	return context.WithValue(ctx, userEmailerKey{}, emailer)
}

// UserEmailerFromContext retrieves the UserEmailer from the context.
func UserEmailerFromContext(ctx context.Context) (UserEmailer, bool) {
	v, ok := ctx.Value(userEmailerKey{}).(UserEmailer)
	return v, ok
}

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

// StartTime returns the start time of the context (if set).
func StartTime(ctx context.Context) (time.Time, bool) {
	v, ok := ctx.Value(loggingKey).(*LoggingContext)
	if !ok {
		return time.Time{}, false
	}
	return v.StartTime, ok
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

// WithLevel forces a log level for the current request/context.
// Level may still be upgraded to Error if an error is present.
func WithLevel(ctx context.Context, level slog.Level) context.Context {
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
	ForceLevel *slog.Level
}

func (l *LoggingContext) SetForceLevel(level slog.Level) {
	l.l.Lock()
	defer l.l.Unlock()
	l.ForceLevel = &level
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
func (l *LoggingContext) Log(ctx context.Context, logger *slog.Logger) {
	l.l.Lock()
	defer l.l.Unlock()

	var lvl slog.Level
	switch {
	case l.setLevelError():
		lvl = slog.LevelError
	case l.ForceLevel != nil:
		lvl = *l.ForceLevel
	default:
		lvl = slog.LevelDebug
	}

	attrs := make([]slog.Attr, 0, 8+len(l.Extras)/2)

	if !l.SkipUser {
		loggedInUser := "unauthenticated"
		if emailer, ok := UserEmailerFromContext(ctx); ok {
			loggedInUser = emailer.Email()
		}
		attrs = append(attrs, slog.String("user", loggedInUser))
	}

	requestMethod, _ := ctx.Value(kithttp.ContextKeyRequestMethod).(string)
	requestURI, _ := ctx.Value(kithttp.ContextKeyRequestURI).(string)
	attrs = append(attrs,
		slog.String("method", requestMethod),
		slog.String("uri", requestURI),
		slog.Duration("took", time.Since(l.StartTime)),
	)

	if len(l.Extras) > 0 {
		for i := 0; i < len(l.Extras)-1; i += 2 {
			key, ok := l.Extras[i].(string)
			if !ok {
				continue
			}
			attrs = append(attrs, slog.Any(key, l.Extras[i+1]))
		}
	}

	if len(l.Errs) > 0 {
		var (
			errs         string
			internalErrs string
			uuids        []string
		)
		separator := " || "
		for _, err := range l.Errs {
			var ewi platform_http.ErrWithInternal
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
			var ewuuid platform_http.ErrorUUIDer
			if errors.As(err, &ewuuid) {
				if uuid := ewuuid.UUID(); uuid != "" {
					uuids = append(uuids, uuid)
				}
			}
		}
		if len(errs) > 0 {
			attrs = append(attrs, slog.String("err", errs))
		}
		if len(internalErrs) > 0 {
			attrs = append(attrs, slog.String("internal", internalErrs))
		}
		if len(uuids) > 0 {
			attrs = append(attrs, slog.String("uuid", strings.Join(uuids, ",")))
		}
	}

	logger.LogAttrs(ctx, lvl, "", attrs...)
}

func (l *LoggingContext) setLevelError() bool {
	if len(l.Errs) == 0 {
		return false
	}

	if len(l.Errs) == 1 {
		var ew platform_http.ErrWithIsClientError
		if errors.As(l.Errs[0], &ew) && ew.IsClientError() {
			return false
		}
	}

	return true
}
