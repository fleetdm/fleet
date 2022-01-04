package table

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kolide/osquery-go"
	"github.com/kolide/osquery-go/plugin/table"
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
	"github.com/rs/zerolog/log"
)

// Runner wraps the osquery extension manager with okglog/run Execute and Interrupt functions.
type Runner struct {
	socket string
	srv    *osquery.ExtensionManagerServer
	cancel func()
}

// NewRunner creates an extension runner.
func NewRunner(socket string) *Runner {
	r := &Runner{socket: socket}
	return r
}

// Execute creates an osquery extension manager server and registers osquery plugins.
func (r *Runner) Execute() error {
	if err := waitForSocket(r.socket, 3*time.Second); err != nil {
		return err
	}

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

	plugins := []osquery.OsqueryPlugin{
		table.NewPlugin("puppet_info", puppet.PuppetInfoColumns(), puppet.PuppetInfoGenerate),
		table.NewPlugin("puppet_logs", puppet.PuppetLogsColumns(), puppet.PuppetLogsGenerate),
		table.NewPlugin("puppet_state", puppet.PuppetStateColumns(), puppet.PuppetStateGenerate),
		table.NewPlugin("google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns(), chromeuserprofiles.GoogleChromeProfilesGenerate),
		table.NewPlugin("file_lines", fileline.FileLineColumns(), fileline.FileLineGenerate),
	}
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

	if r.srv != nil {
		r.srv.Shutdown(context.Background())
	}
}

// waitForSocket waits for the osquery socket to exist.
//
// This method was copied from osquery-go because we can't rely on option.ServerTimeout.
// Such timeout is used both for waiting for the socket and for the thrift transport.
func waitForSocket(sockPath string, timeout time.Duration) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			switch _, err := os.Stat(sockPath); {
			case err == nil:
				return nil
			case os.IsNotExist(err):
				continue
			default:
				return fmt.Errorf("stat socket %s failed: %w", sockPath, err)
			}
		}
	}
}
