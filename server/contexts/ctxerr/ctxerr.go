// Package ctxerr provides functions to wrap errors with annotations and
// stack traces, and to handle those errors such that unique instances of
// those errors will be stored for an amount of time so that it can be
// used to troubleshoot issues.
//
// Typical uses of this package should be to call New or Wrap[f] as close as
// possible from where the error is encountered (or where it needs to be
// created for New), and then to call Handle with the error only once, after it
// bubbled back to the top of the call stack (e.g. in the HTTP handler, or in
// the CLI command, etc.). It is fine to wrap the error with more annotations
// along the way, by calling Wrap[f].
package ctxerr

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/pkg/errors"
	"github.com/rotisserie/eris"
)

type key int

const errHandlerKey key = 0

// NewContext returns a context derived from ctx that contains the provided
// error handler.
func NewContext(ctx context.Context, eh *errorstore.Handler) context.Context {
	return context.WithValue(ctx, errHandlerKey, eh)
}

func fromContext(ctx context.Context) *errorstore.Handler {
	v, _ := ctx.Value(errHandlerKey).(*errorstore.Handler)
	return v
}

// New creates a new error with the provided error message.
func New(ctx context.Context, errMsg string) error {
	return ensureCommonMetadata(ctx, errors.New(errMsg))
}

// Wrap annotates err with the provided message.
func Wrap(ctx context.Context, err error, msg string) error {
	err = ensureCommonMetadata(ctx, err)
	// do not wrap with eris.Wrap, as we want only the root error closest to the
	// actual error condition to capture the stack trace, others just wrap using
	// pkg/errors.
	return errors.Wrap(err, msg)
}

// Wrapf annotates err with the provided formatted message.
func Wrapf(ctx context.Context, err error, fmsg string, args ...interface{}) error {
	err = ensureCommonMetadata(ctx, err)
	// do not wrap with eris.Wrap, as we want only the root error closest to the
	// actual error condition to capture the stack trace, others just wrap using
	// pkg/errors.
	return errors.Wrapf(err, fmsg, args...)
}

// Handle handles err by passing it to the registered error handler,
// deduplicating it and storing it for a configured duration.
func Handle(ctx context.Context, err error) error {
	if eh := fromContext(ctx); eh != nil {
		return eh.Store(ctx, err)
	}
	return err
}

func ensureCommonMetadata(ctx context.Context, err error) error {
	var sf interface{ StackFrames() []uintptr }
	if err != nil && !errors.As(err, &sf) {
		// no eris error nowhere in the chain, add the common metadata with the stack trace
		// TODO: more metadata from ctx: user, host, etc.
		err = eris.Wrapf(err, "timestamp: %s", time.Now().Format(time.RFC3339))
	}
	return err
}
