package table

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptoinfotable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/firefox_preferences"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/fleetd_logs"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/sntp_request"
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Runner wraps the osquery extension manager with okglog/run Execute and Interrupt functions.
type Runner struct {
	socket          string
	tableExtensions []Extension
	executeDone     chan struct{}

	// mu protects access to srv, ctx and cancel in Execute and Interrupt.
	mu     sync.Mutex
	srv    *osquery.ExtensionManagerServer
	ctx    context.Context
	cancel func()
}

// Extension implements a osquery-go table extension.
type Extension interface {
	// Name returns the name of the table.
	Name() string
	// Columns returns the definition of the table columns.
	Columns() []table.ColumnDefinition
	// GenerateFunc generates results for a query.
	GenerateFunc(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error)
}

// Opt allows configuring a Runner.
type Opt func(*Runner)

// PluginOpts provides options required by some tables.
type PluginOpts struct {
	Socket string
}

// WithExtension registers the given Extension on the Runner.
func WithExtension(t Extension) Opt {
	return func(r *Runner) {
		r.tableExtensions = append(r.tableExtensions, t)
	}
}

// NewRunner creates an extension runner.
func NewRunner(socket string, opts ...Opt) *Runner {
	r := &Runner{
		socket:      socket,
		executeDone: make(chan struct{}),
	}
	for _, fn := range opts {
		fn(r)
	}
	return r
}

// Execute creates an osquery extension manager server and registers osquery plugins.
func (r *Runner) Execute() error {
	defer close(r.executeDone)

	if err := waitExtensionSocket(r.socket, 1*time.Minute); err != nil {
		return err
	}

	ctx, _ := r.getContextAndCancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		srv, err := osquery.NewExtensionManagerServer(
			"com.fleetdm.orbit.osquery_extension.v1",
			r.socket,
			// This timeout is only used for registering the extension tables
			// and for the heartbeat ping requests in r.srv.Run().
			//
			// On some systems, registering tables takes more than a couple
			// of seconds, thus set timeout to minutes instead (see #3878).
			osquery.ServerTimeout(5*time.Minute))
		if err == nil {
			r.setSrv(srv)
			ticker.Stop()
			break
		}

		select {
		case <-ticker.C:
			log.Error().Err(err).Msg("NewExtensionManagerServer failed")
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		}
	}

	plugins := OrbitDefaultTables()

	opts := PluginOpts{Socket: r.socket}
	platformTables, err := PlatformTables(opts)
	if err != nil {
		return fmt.Errorf("populating platform tables: %w", err)
	}

	plugins = append(plugins, platformTables...)
	for _, t := range r.tableExtensions {
		plugins = append(plugins, table.NewPlugin(
			t.Name(),
			t.Columns(),
			t.GenerateFunc,
		))
	}
	r.srv.RegisterPlugin(plugins...)

	if err := r.srv.Run(); err != nil {
		return err
	}

	return nil
}

func OrbitDefaultTables() []osquery.OsqueryPlugin {
	plugins := []osquery.OsqueryPlugin{
		// MacAdmins extensions.
		table.NewPlugin("puppet_info", puppet.PuppetInfoColumns(), puppet.PuppetInfoGenerate),
		table.NewPlugin("puppet_logs", puppet.PuppetLogsColumns(), puppet.PuppetLogsGenerate),
		table.NewPlugin("puppet_state", puppet.PuppetStateColumns(), puppet.PuppetStateGenerate),
		table.NewPlugin("google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns(), chromeuserprofiles.GoogleChromeProfilesGenerate),
		table.NewPlugin("file_lines", fileline.FileLineColumns(), fileline.FileLineGenerate),

		// Orbit extensions.
		table.NewPlugin("sntp_request", sntp_request.Columns(), sntp_request.GenerateFunc),
		fleetd_logs.TablePlugin(),

		// Note: the logger passed here and to all other tables is the global logger from zerolog.
		// This logger has already been configured with some required settings in
		// orbit/cmd/orbit/orbit.go.
		firefox_preferences.TablePlugin(log.Logger),
		cryptoinfotable.TablePlugin(log.Logger),

		// Additional data format tables
		dataflattentable.TablePlugin(log.Logger, dataflattentable.JsonType),  // table name is "parse_json"
		dataflattentable.TablePlugin(log.Logger, dataflattentable.JsonlType), // table name is "parse_jsonl"
		dataflattentable.TablePlugin(log.Logger, dataflattentable.XmlType),   // table name is "parse_xml"
		dataflattentable.TablePlugin(log.Logger, dataflattentable.IniType),   // table name is "parse_ini"

	}
	return plugins
}

// Interrupt shuts down the osquery manager server.
func (r *Runner) Interrupt(err error) {
	if _, cancel := r.getContextAndCancel(); cancel != nil {
		cancel()
	}
	<-r.executeDone
	if srv := r.getSrv(); srv != nil {
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Debug().Err(err).Msg("shutdown extension")
		}
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

func (r *Runner) setSrv(s *osquery.ExtensionManagerServer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.srv = s
}

func (r *Runner) getSrv() *osquery.ExtensionManagerServer {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.srv
}

func (r *Runner) getContextAndCancel() (context.Context, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ctx != nil {
		return r.ctx, r.cancel
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel
	return r.ctx, r.cancel
}
