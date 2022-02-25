package table

import (
	"context"
	"errors"
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

	// TODO(lucas): Check if set/get of these two should be protected.
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
	log.Debug().Msg("start osquery extension")

	if err := waitExtensionSocket(r.socket, 1*time.Minute); err != nil {
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
			// This timeout is only used for registering the extension tables
			// and for the heartbeat ping requests in r.srv.Run().
			//
			// On some systems, registering tables takes more than a couple
			// of seconds, thus set timeout to minutes instead (see #3878).
			osquery.ServerTimeout(5*time.Minute))
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
	log.Debug().Err(err).Msg("interrupt osquery extension")
	if r.cancel != nil {
		r.cancel()
	}
	if r.srv != nil {
		r.srv.Shutdown(context.Background())
	}
}

// waitExtensionSocket waits until the osquery extension manager socket is ready.
// First, it waits for the unix socket/Windows pipe to be available.
// Then, it tries connecting as a client and sending a ping to ensure that osquery
// is ready for extensions.
//
// This method is a workaround for https://github.com/osquery/osquery-go/issues/80.
func waitExtensionSocket(sockPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := waitSocketExists(ctx, sockPath); err != nil {
		return err
	}
	if err := waitSocketReady(ctx, sockPath); err != nil {
		return err
	}
	return nil
}

func waitSocketExists(ctx context.Context, sockPath string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("extension socket stat timeout")
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

func waitSocketReady(ctx context.Context, sockPath string) error {
	serverClient, err := osquery.NewClient(sockPath, 3*time.Second)
	if err != nil {
		return err
	}
	defer serverClient.Close()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("extension socket ping timeout")
		case <-ticker.C:
			status, err := serverClient.Ping()
			if err != nil {
				log.Debug().Err(err).Msgf(
					"failed to ping extension socket, retrying...",
				)
				continue
			}
			if status.Code == 0 {
				log.Debug().Msg("extension manager checked")
				return nil
			}
			log.Debug().Int32("status_code", status.Code).Str("status_message", status.Message).Msgf(
				"extension socket not ready, retrying...",
			)
		}
	}
}
