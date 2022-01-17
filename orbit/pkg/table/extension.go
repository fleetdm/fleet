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
	log.Debug().Err(err).Msg("interrupt osquery extension")
	r.cancel()

	if r.srv != nil {
		r.srv.Shutdown(context.Background())
	}
}

// waitExtensionSocket waits until the osquery extension manager socket is ready.
//
// We can't rely on osquery-go's option.ServerTimeout. Such timeout is used both for waiting
// for the socket and for the thrift transport.
func waitExtensionSocket(sockPath string, timeout time.Duration) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
FOR_LOOP:
	for {
		select {
		case <-ctx.Done():
			return errors.New("extension socket stat timeout")
		case <-ticker.C:
			switch _, err := os.Stat(sockPath); {
			case err == nil:
				break FOR_LOOP
			case os.IsNotExist(err):
				continue
			default:
				return fmt.Errorf("stat socket %s failed: %w", sockPath, err)
			}
		}
	}

	serverClient, err := osquery.NewClient(sockPath, 3*time.Second)
	if err != nil {
		return err
	}
	defer serverClient.Close()

	for {
		status, err := serverClient.Ping()
		if err == nil && status.Code == 0 {
			log.Debug().Msg("extension manager checked")
			return nil
		}
		log.Debug().Msgf(
			"extension socket ping failed: {code: %d, message: %s, err: %s}, retrying...",
			status.Code, status.Message, err,
		)
		select {
		case <-ctx.Done():
			return errors.New("extension socket ping timeout")
		case <-ticker.C:
			// OK
		}
	}
}
