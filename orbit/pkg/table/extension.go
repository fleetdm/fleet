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

// NewRunner creates an extension runner.
func NewRunner(socket string) (*Runner, error) {
	r := &Runner{socket: socket}
	return r, nil
}

// Execute creates an osquery extension manager server and registers osquery plugins.
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
			ticker.Stop()
			break
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		}
	}

	var plugins []osquery.OsqueryPlugin
	plugins = append(plugins, platformTables()...)
	r.srv.RegisterPlugin(plugins...)

	if err := r.srv.Run(); err != nil {
		return err
	}

	return nil
}

// Interrupt shuts down the osquery manager server.
func (r *Runner) Interrupt(err error) {
	log.Debug().Msg("interrupt osquery extension")
	r.cancel()

	r.srv.Shutdown(context.Background())
}
