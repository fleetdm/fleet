package errors

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
)

var errCh chan error

type errorHandler struct {
	pool   fleet.RedisPool
	logger kitlog.Logger
}

func StartErrorHandler(ctx context.Context, pool fleet.RedisPool, logger kitlog.Logger) {
	errCh = make(chan error)

	e := &errorHandler{
		pool:   pool,
		logger: logger,
	}
	go e.handleErrors(ctx)
}

func (e errorHandler) handleErrors(ctx context.Context) {
	conn := e.pool.Get()
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			_ = err
			err := conn.Send("SET", sqlKey, sql, "EX", queryExpiration.Seconds())
		}
	}
}

func New(err error) error {
	ticker := time.NewTicker(5 * time.Second)
	select {
	case errCh <- err:
	case <-ticker.C:
	}
	return err
}
