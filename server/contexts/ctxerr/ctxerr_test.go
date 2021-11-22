package ctxerr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	kitlog "github.com/go-kit/kit/log"
	pkgerrors "github.com/pkg/errors" //nolint:depguard
	"github.com/stretchr/testify/require"
)

func TestCause(t *testing.T) {
	ctx := context.Background()
	eh := errorstore.NewHandler(ctx, redistest.NopRedis(), kitlog.NewNopLogger(), time.Minute)
	ctx = NewContext(ctx, eh)

	errNew := errors.New("new")
	fmtWrap := fmt.Errorf("fmt: %w", errNew)
	pkgWrap := pkgerrors.Wrap(errNew, "pkg")
	pkgFmtWrap := pkgerrors.Wrap(fmtWrap, "pkg")
	fmtPkgWrap := fmt.Errorf("fmt: %w", pkgWrap)
	ctxNew := New(ctx, "ctxerr")        // this returns an eris error that wraps a standard error
	ctxNewRoot := errors.Unwrap(ctxNew) // this gets the standard error wrapped in ctxNew
	ctxWrap := Wrap(ctx, ctxNew, "wrap")
	ctxDoubleWrap := Wrap(ctx, ctxWrap, "re-wrap")
	pkgFmtCtxWrap := pkgerrors.Wrap(fmt.Errorf("fmt: %w", ctxWrap), "pkg")
	fmtPkgCtxWrap := fmt.Errorf("fmt: %w", pkgerrors.Wrap(ctxWrap, "pkg"))

	cases := []struct {
		in, out error
	}{
		{nil, nil},
		{io.EOF, io.EOF},
		{errNew, errNew},
		{fmtWrap, errNew},
		{pkgWrap, errNew},
		{pkgFmtWrap, errNew},
		{fmtPkgWrap, errNew},
		{ctxNew, ctxNewRoot},
		{ctxWrap, ctxNewRoot},
		{ctxDoubleWrap, ctxNewRoot},
		{pkgFmtCtxWrap, ctxNewRoot},
		{fmtPkgCtxWrap, ctxNewRoot},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%T: %[1]v", c.in), func(t *testing.T) {
			got := Cause(c.in)
			require.Equal(t, c.out, got)
		})
	}
}
