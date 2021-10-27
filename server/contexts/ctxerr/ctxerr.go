package ctxerr

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/errors"
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

func WrapAndHandle(ctx context.Context, err error, msg string) error {
	panic("unimplemented")
}

func WrapfAndHandle(ctx context.Context, err error, msg string, args ...interface{}) error {
	panic("unimplemented")
}

func Wrap(ctx context.Context, err error, msg string) error {
	panic("unimplemented")
}

func Wrapf(ctx context.Context, err error, fmsg string, args ...interface{}) error {
	panic("unimplemented")
}

func Handle(ctx context.Context, err error) error {
	if eh := fromContext(ctx); eh != nil {
		return eh.New(ctx, err)
	}
	return err
}
