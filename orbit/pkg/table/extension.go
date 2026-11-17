package table

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_agent_sessions"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptoinfotable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/filecontents"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/firefox_preferences"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/fleetd_logs"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/go_binaries"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/mcp_listening_servers"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/netskope"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/sentinelone"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/sntp_request"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/yaml_to_json"
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
	"github.com/osquery/osquery-go"
	osquerygen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// ExtensionName is the name orbit's extension manager server registers with
// osquery. It is what osquery's --extensions_require flag takes to wait for
// this extension before activating plugins.
const ExtensionName = "com.fleetdm.orbit.osquery_extension.v1"

// Runner wraps the osquery extension manager with okglog/run Execute and Interrupt functions.
type Runner struct {
	socket          string
	tableExtensions []Extension
	plugins         []osquery.OsqueryPlugin
	executeDone     chan struct{}

	// newClient creates a client to osquery's extension manager.
	newClient func(timeout time.Duration) (osquery.ExtensionManager, error)
	// checkInterval is how often the runner verifies osquery still lists the
	// extension as registered.
	checkInterval time.Duration
	// retryBackoff is the delay before the first re-registration attempt; it
	// doubles on each consecutive failure.
	retryBackoff time.Duration

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

// WithPlugin registers an arbitrary osquery plugin (e.g. a distributed plugin)
// on the Runner, alongside the table plugins.
func WithPlugin(p osquery.OsqueryPlugin) Opt {
	return func(r *Runner) {
		r.plugins = append(r.plugins, p)
	}
}

// NewRunner creates an extension runner.
func NewRunner(socket string, opts ...Opt) *Runner {
	r := &Runner{
		socket:        socket,
		executeDone:   make(chan struct{}),
		checkInterval: 10 * time.Second,
		retryBackoff:  1 * time.Second,
	}
	r.newClient = func(timeout time.Duration) (osquery.ExtensionManager, error) {
		return osquery.NewClient(r.socket, timeout)
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

	opts := PluginOpts{Socket: r.socket}
	plugins := OrbitDefaultTables(opts)
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
	plugins = append(plugins, r.plugins...)

	return r.run(ctx, plugins, func(client osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error) {
		return osquery.NewExtensionManagerServer(ExtensionName, r.socket, osquery.WithClient(client))
	})
}

const (
	// serverClientTimeout is the timeout of the client the extension manager
	// server uses to register the extension and ping osquery. On some systems,
	// registering tables takes more than a couple of seconds (see #3878).
	serverClientTimeout = 5 * time.Minute
	// maxConsecutiveSessionFailures is how many extension sessions in a row can
	// end in an error before the runner gives up and returns the error, which
	// restarts orbit.
	maxConsecutiveSessionFailures = 5
	// healthySessionDuration is how long a session must last to reset the
	// consecutive failure count.
	healthySessionDuration = 1 * time.Minute
	maxRetryBackoff        = 30 * time.Second
)

var errExtensionNotRegistered = errors.New("osquery no longer lists the extension as registered")

// run registers plugins and serves them until osquery goes away.
//
// If osquery rejects the registration because another extension (e.g. a
// third-party one loaded via --extensions_autoload) already registered an item
// with the same name, the conflicting plugin is dropped and registration is
// retried, so that a name collision doesn't take orbit down.
//
// If the session is lost (osquery deregisters the extension, or the
// connection to osquery breaks), the extension is registered again in place
// rather than returning an error, which would restart all of orbit.
func (r *Runner) run(ctx context.Context, plugins []osquery.OsqueryPlugin, newServer func(osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error)) error {
	failures := 0
	for {
		srv, client, err := r.newServer(ctx, newServer)
		if err != nil {
			return err
		}
		srv.RegisterPlugin(plugins...)

		started := time.Now()
		err = r.serve(ctx, srv, client)
		if err == nil || ctx.Err() != nil {
			// A nil error means osquery asked the extension to shut down.
			return err
		}
		if name, ok := registryConflictItem(err); ok {
			// Only tables are dropped; losing another kind of plugin (e.g. distributed)
			// would silently break orbit, so that conflict stays fatal.
			remaining := slices.DeleteFunc(slices.Clone(plugins), func(p osquery.OsqueryPlugin) bool {
				return p.RegistryName() == "table" && p.Name() == name
			})
			if len(remaining) == len(plugins) {
				// The conflicting item isn't one of our tables, retrying won't help.
				return err
			}
			plugins = remaining
			log.Warn().Str("table", name).Msg("skipping fleetd table: another osquery extension already registered an item with the same name")
			continue
		}

		if time.Since(started) >= healthySessionDuration {
			failures = 0
		}
		failures++
		if failures > maxConsecutiveSessionFailures {
			return fmt.Errorf("osquery extension session failed %d consecutive times: %w", failures, err)
		}
		// When the session ends because the connection broke, osquery-go's
		// deregistration goes over that broken connection and can fail, leaving a
		// stale registration that makes osquery reject the new one as a duplicate.
		if uuid, ok := client.registeredUUID(); ok {
			r.deregister(uuid)
		}
		backoff := min(r.retryBackoff<<(failures-1), maxRetryBackoff)
		log.Error().Err(err).Int("attempt", failures).Dur("retry_in", backoff).Msg("osquery extension session ended, registering extension again")
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return err
		}
	}
}

// serve runs srv until it stops, osquery no longer lists the extension as
// registered, or ctx is canceled.
//
// osquery-go only pings the extension manager, which proves osqueryd is alive
// but not that this extension is still registered with it, so the
// registration is checked separately.
func (r *Runner) serve(ctx context.Context, srv *osquery.ExtensionManagerServer, client *registrationClient) error {
	lost := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)

	go func() {
		switch err := r.watchRegistration(ctx, done, client); {
		case err != nil:
			lost <- err
		case ctx.Err() == nil:
			// srv.Run already returned.
			return
		}
		if err := srv.Shutdown(context.WithoutCancel(ctx)); err != nil {
			log.Debug().Err(err).Msg("shutdown extension")
		}
	}()

	err := srv.Run()
	// srv doesn't close a client passed with osquery.WithClient.
	client.Close()
	select {
	case lostErr := <-lost:
		return lostErr
	default:
		return err
	}
}

// watchRegistration returns errExtensionNotRegistered once osquery no longer
// lists the extension registered by client, and nil when done is closed or ctx
// is canceled.
func (r *Runner) watchRegistration(ctx context.Context, done <-chan struct{}, client *registrationClient) error {
	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()

	var checkClient osquery.ExtensionManager
	defer func() {
		if checkClient != nil {
			checkClient.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-done:
			return nil
		case <-ticker.C:
		}

		uuid, ok := client.registeredUUID()
		if !ok {
			continue
		}
		if checkClient == nil {
			c, err := r.newClient(r.checkInterval)
			if err != nil {
				log.Debug().Err(err).Msg("connect to osquery to check extension registration")
				continue
			}
			checkClient = c
		}
		extensions, err := checkClient.Extensions()
		if err != nil {
			// Losing the connection to osquery itself is detected by osquery-go's ping.
			log.Debug().Err(err).Msg("check extension registration")
			checkClient.Close()
			checkClient = nil
			continue
		}
		if _, ok := extensions[uuid]; !ok {
			return errExtensionNotRegistered
		}
	}
}

// deregister deregisters the extension with the given UUID, if osquery still
// has it registered.
func (r *Runner) deregister(uuid osquerygen.ExtensionRouteUUID) {
	client, err := r.newClient(r.checkInterval)
	if err != nil {
		log.Debug().Err(err).Msg("connect to osquery to deregister stale extension")
		return
	}
	defer client.Close()

	extensions, err := client.Extensions()
	if err != nil {
		log.Debug().Err(err).Msg("list osquery extensions")
		return
	}
	if _, ok := extensions[uuid]; !ok {
		return
	}
	log.Info().Int64("uuid", int64(uuid)).Msg("deregistering stale osquery extension")
	if _, err := client.DeregisterExtension(uuid); err != nil {
		log.Debug().Err(err).Msg("deregister stale osquery extension")
	}
}

// registrationClient records the UUID osquery assigns to the extension, which
// osquery-go's ExtensionManagerServer doesn't expose.
type registrationClient struct {
	osquery.ExtensionManager

	mu         sync.Mutex
	uuid       osquerygen.ExtensionRouteUUID
	registered bool
}

func (c *registrationClient) RegisterExtension(info *osquerygen.InternalExtensionInfo, registry osquerygen.ExtensionRegistry) (*osquerygen.ExtensionStatus, error) {
	stat, err := c.ExtensionManager.RegisterExtension(info, registry)
	if err == nil && stat != nil && stat.Code == 0 {
		c.mu.Lock()
		c.uuid, c.registered = stat.UUID, true
		c.mu.Unlock()
	}
	return stat, err
}

func (c *registrationClient) registeredUUID() (osquerygen.ExtensionRouteUUID, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.uuid, c.registered
}

// newServer creates the extension manager server, retrying until it succeeds
// or ctx is canceled.
func (r *Runner) newServer(ctx context.Context, newServer func(osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error)) (*osquery.ExtensionManagerServer, *registrationClient, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		srv, client, err := r.tryNewServer(newServer)
		if err == nil {
			r.setSrv(srv)
			return srv, client, nil
		}

		select {
		case <-ticker.C:
			log.Error().Err(err).Msg("NewExtensionManagerServer failed")
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
}

func (r *Runner) tryNewServer(newServer func(osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error)) (*osquery.ExtensionManagerServer, *registrationClient, error) {
	c, err := r.newClient(serverClientTimeout)
	if err != nil {
		return nil, nil, err
	}
	client := &registrationClient{ExtensionManager: c}
	srv, err := newServer(client)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	return srv, client, nil
}

// registryConflictRegexp matches the error osquery returns when an extension
// registers an item already registered by another extension.
var registryConflictRegexp = regexp.MustCompile(`Registry item (\S+) conflicts with another item of the same name`)

func registryConflictItem(err error) (string, bool) {
	m := registryConflictRegexp.FindStringSubmatch(err.Error())
	if m == nil {
		return "", false
	}
	return m[1], true
}

func OrbitDefaultTables(opts PluginOpts) []osquery.OsqueryPlugin {
	plugins := []osquery.OsqueryPlugin{
		// MacAdmins extensions.
		table.NewPlugin("puppet_info", puppet.PuppetInfoColumns(), puppet.PuppetInfoGenerate),
		table.NewPlugin("puppet_logs", puppet.PuppetLogsColumns(), puppet.PuppetLogsGenerate),
		table.NewPlugin("puppet_state", puppet.PuppetStateColumns(), puppet.PuppetStateGenerate),
		table.NewPlugin("google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns(), chromeuserprofiles.GoogleChromeProfilesGenerate),
		table.NewPlugin("file_lines", fileline.FileLineColumns(), fileline.FileLineGenerate),
		table.NewPlugin("file_contents", filecontents.Columns(), filecontents.Generate),

		// Orbit extensions.
		table.NewPlugin("sntp_request", sntp_request.Columns(), sntp_request.GenerateFunc),
		fleetd_logs.TablePlugin(),
		table.NewPlugin("go_binaries", go_binaries.GoBinariesColumns(), go_binaries.GenerateGoBinaries),

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

		// mcp_listening_servers: lists running processes from core processes table via osquery client
		table.NewPlugin(
			"mcp_listening_servers",
			mcp_listening_servers.Columns(),
			func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
				return mcp_listening_servers.Generate(ctx, queryContext, opts.Socket)
			},
		),

		table.NewPlugin("yaml_to_json", yaml_to_json.Columns(), yaml_to_json.GenerateFunc),

		// ai_tools: unified table of AI software (apps, IDE plugins, agents, MCP
		// servers, sockets, instruction files, browser extensions) and their risk
		// factors. Vendored from github.com/karmine05/agentic-detector.
		table.NewPlugin("ai_tools", ai_tools.Columns(), ai_tools.Generate),

		// ai_agent_sessions / ai_agent_session_messages: sessions of AI coding
		// agents (Claude Code, Copilot CLI, Cursor CLI, Codex CLI, Gemini CLI)
		// and their content, from every user's transcripts. Ported from
		// github.com/allenhouchins/fleet-extensions.
		table.NewPlugin("ai_agent_sessions", ai_agent_sessions.SessionsColumns(), ai_agent_sessions.GenerateSessions),
		table.NewPlugin("ai_agent_session_messages", ai_agent_sessions.MessagesColumns(), ai_agent_sessions.GenerateMessages),

		// netskope: local Netskope client state, from `nsdiag -f`.
		// Registered on every platform: hosts without the client report a single
		// row with client_installed = 0.
		table.NewPlugin("netskope", netskope.Columns(), netskope.Generate),

		// sentinelone: local SentinelOne agent state, from `sentinelctl status`.
		// Registered on every platform: hosts without the agent report no rows.
		table.NewPlugin("sentinelone", sentinelone.Columns(), sentinelone.Generate),
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
