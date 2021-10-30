package table

import (
	"context"
	"time"

	"github.com/kolide/osquery-go"
	"github.com/rs/zerolog/log"
)

// Runner wraps the osquery extension manager with okglog/run Execute and Interrupt functions.
type Runner struct {
	socket string
	srv    *osquery.ExtensionManagerServer
	cancel func()
}

func NewRunner(socket string) (*Runner, error) {
	r := &Runner{socket: socket}
	return r, nil
}

func (r *Runner) Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	var err error
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		r.srv, err = osquery.NewExtensionManagerServer(
			"com.fleetdm.orbit.osquery_extension.v1",
			r.socket,
			osquery.ServerTimeout(3*time.Second))
		if err == nil {
			break
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var plugins []osquery.OsqueryPlugin
	for _, t := range platformTables() {
		plugins = append(plugins, t)
	}
	r.srv.RegisterPlugin(plugins...)

	if err := r.srv.Run(); err != nil {
		return err
	}

	return nil

}

func (r *Runner) Interrupt(err error) {
	log.Debug().Msg("interrupt osquery")
	r.cancel()

	r.srv.Shutdown(context.Background())
}
