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

	goerrors "errors"

	"github.com/fleetdm/fleet/v4/server/errors"
	"github.com/rotisserie/eris"
)

type key int

const errHandlerKey key = 0

// NewContext returns a context derived from ctx that contains the provided
// error handler.
func NewContext(ctx context.Context, eh *errors.Handler) context.Context {
	return context.WithValue(ctx, errHandlerKey, eh)
}

func fromContext(ctx context.Context) *errors.Handler {
	v, _ := ctx.Value(errHandlerKey).(*errors.Handler)
	return v
}

// TODO: double-check the eris stack trace, the location used for hashing
// will end up always being this call, from what I can tell (eris does not
// support passing the number of caller frames to skip).

// New creates a new error with the provided error message.
func New(ctx context.Context, errMsg string) error {
	return ensureCommonMetadata(ctx, goerrors.New(errMsg))
}

// Wrap annotates err with the provided message.
func Wrap(ctx context.Context, err error, msg string) error {
	err = ensureCommonMetadata(ctx, err)
	return eris.Wrap(err, msg)
}

// Wrapf annotates err with the provided formatted message.
func Wrapf(ctx context.Context, err error, fmsg string, args ...interface{}) error {
	err = ensureCommonMetadata(ctx, err)
	return eris.Wrapf(err, fmsg, args...)
}

// Handle handles err by passing it to the registered error handler,
// deduplicating it and storing it for a configured duration.
func Handle(ctx context.Context, err error) error {
	if eh := fromContext(ctx); eh != nil {
		return eh.New(ctx, err)
	}
	return err
}

func ensureCommonMetadata(ctx context.Context, err error) error {
	var sf interface{ StackFrames() []uintptr }
	if err != nil && !goerrors.As(err, &sf) {
		// no eris error nowhere in the chain, add the common metadata
		// TODO: more metadata from ctx: user, host, etc.
		err = eris.Wrapf(err, "timestamp: %s", time.Now().Format(time.RFC3339))
	}
	return err
}
