package wstransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/config"
	"github.com/rs/zerolog/log"
)

// ConfigPluginName is the plugin name osquery must be pointed at via
// --config_plugin to get its config from orbit.
const ConfigPluginName = "fleet_orbit_config"

// LocalConfigRefresh is the interval, in seconds, at which osquery asks orbit
// for its config (osquery's --config_refresh, a command-line-only flag). The
// request is a localhost call answered from memory, so it can be short: it
// bounds how long a fetched config waits to be applied. osquery skips
// reloading an unchanged config.
const LocalConfigRefresh = 10

// configSourceName is the name osquery's TLS config plugin gives its config
// source. Keeping it means switching plugins doesn't look like a new source
// to osquery.
const configSourceName = "tls_plugin"

// DefaultServerConfigRefresh is the server poll interval while disconnected
// when the osquery flagfile doesn't set --config_refresh: fleetd's default.
const DefaultServerConfigRefresh = 60 * time.Second

// DefaultOsqueryConfigPollInterval is the fallback poll interval of connected
// agents until the server sends one.
const DefaultOsqueryConfigPollInterval = 10 * time.Minute

// disconnectSpread spreads the first fetch after a disconnect: a server
// restart disconnects every agent at once.
const disconnectSpread = 30 * time.Second

// configUnchangedETag is the etag of the server's response to a current etag,
// {"etag":"ok"}.
const configUnchangedETag = "ok"

// configClient is the subset of client.OrbitClient used by ConfigFetcher.
type configClient interface {
	OsqueryConfig(etag string, fallbackChange bool) (json.RawMessage, error)
}

// ConfigFetcher serves osquery its config (as its config plugin) from memory,
// fetching it from the server when nudged, on a slow fallback timer while the
// WebSocket is connected, and at osquery's own config refresh while it isn't
// (see NewConfigFetcher).
type ConfigFetcher struct {
	client    configClient
	connected func() bool
	now       func() time.Time
	// fetched is notified after each fetch attempt, once the fetcher's state
	// reflects it, for tests.
	fetched func()

	// firstFetchMu serializes the synchronous fetches made until a config is
	// cached.
	firstFetchMu sync.Mutex

	mu sync.Mutex
	// config is the osquery-ready config, empty until the first fetch.
	config        string
	etag          string
	serverRefresh time.Duration
	pollInterval  time.Duration
	nextFetch     time.Time
	// nextIsFallback reports whether the next fetch is the fallback poll of a
	// connected agent.
	nextIsFallback bool
	fetching       bool
	// pending is set when a nudge arrives during a fetch: the change may have
	// been made after the fetch read the config.
	pending bool
	// reportFallbackChange is set when a fallback poll found the config
	// changed, to report on the next fetch.
	reportFallbackChange bool
	// firstFetchErr is the error of the last failed fetch before a config was
	// cached, returned to osquery until the retry is due.
	firstFetchErr error
}

// NewConfigFetcher returns a ConfigFetcher polling the server every
// serverRefresh while the WebSocket is disconnected: osquery's config refresh
// before orbit served its config (see ServerConfigRefresh).
func NewConfigFetcher(client configClient, connected func() bool, serverRefresh time.Duration) *ConfigFetcher {
	return &ConfigFetcher{
		client:        client,
		connected:     connected,
		now:           time.Now,
		serverRefresh: serverRefresh,
		pollInterval:  DefaultOsqueryConfigPollInterval,
	}
}

// NewConfigPlugin returns the osquery config plugin serving f's config.
func NewConfigPlugin(f *ConfigFetcher) *config.Plugin {
	return config.NewPlugin(ConfigPluginName, func(ctx context.Context) (map[string]string, error) {
		cfg, err := f.GenConfig()
		if err != nil {
			return nil, err
		}
		return map[string]string{configSourceName: cfg}, nil
	})
}

// SetPollInterval sets the fallback poll interval of a connected agent, as
// directed by the server.
func (f *ConfigFetcher) SetPollInterval(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	before := f.fallbackIntervalLocked()
	f.pollInterval = d
	// Bring a scheduled fallback poll forward to the shorter interval (the
	// fetcher starts with the default one, before the server's arrives).
	if delta := f.fallbackIntervalLocked() - before; f.nextIsFallback && delta < 0 {
		f.nextFetch = f.nextFetch.Add(delta)
	}
}

// GenConfig returns the config for osquery. The first call fetches it;
// later calls return the cached config and start a fetch in the background
// when one is due, which osquery picks up on its next call.
//
// osquery retries a failed call after its config_accelerated_refresh, not
// its config_refresh.
func (f *ConfigFetcher) GenConfig() (string, error) {
	f.mu.Lock()
	if f.config == "" {
		f.mu.Unlock()
		return f.firstFetch()
	}
	defer f.mu.Unlock()
	if !f.fetching && !f.now().Before(f.nextFetch) {
		f.startFetchLocked(f.nextIsFallback)
	}
	return f.config, nil
}

// firstFetch fetches the config synchronously, for GenConfig until a config
// is cached. After a failure, it returns the failure until the retry is due:
// osquery's config_accelerated_refresh may be shorter.
func (f *ConfigFetcher) firstFetch() (string, error) {
	f.firstFetchMu.Lock()
	defer f.firstFetchMu.Unlock()

	f.mu.Lock()
	if f.config != "" {
		// Fetched by a concurrent call.
		defer f.mu.Unlock()
		return f.config, nil
	}
	if f.firstFetchErr != nil && f.now().Before(f.nextFetch) {
		defer f.mu.Unlock()
		return "", f.firstFetchErr
	}
	f.fetching = true
	f.mu.Unlock()

	err := f.fetch(false)

	f.mu.Lock()
	f.fetching = false
	f.firstFetchErr = err
	if f.pending {
		// Nudged during the fetch, which may have read the config before the
		// change.
		f.pending = false
		if f.config != "" {
			f.startFetchLocked(false)
		}
	}
	cfg := f.config
	f.mu.Unlock()
	f.notifyFetched()
	if err != nil {
		return "", err
	}
	return cfg, nil
}

// Nudge fetches the config now: the server notified that it may have changed.
// Non-blocking.
func (f *ConfigFetcher) Nudge() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.config == "" && !f.fetching {
		// The first fetch, on osquery's first call, gets the current config.
		return
	}
	f.startFetchLocked(false)
}

// Disconnected brings the next fetch forward: without the WebSocket, changes
// are no longer nudged.
func (f *ConfigFetcher) Disconnected() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if next := f.now().Add(rand.N(disconnectSpread)); next.Before(f.nextFetch) { //nolint:gosec // jitter does not need cryptographic randomness
		f.nextFetch = next
		f.nextIsFallback = false
	}
}

// startFetchLocked starts a background fetch, or marks one pending if a fetch
// is in flight (only nudges do: a due fetch waits for the one in flight to
// reschedule). fallback reports whether it is the fallback poll. f.mu must be
// held.
func (f *ConfigFetcher) startFetchLocked(fallback bool) {
	if f.fetching {
		f.pending = true
		return
	}
	f.fetching = true
	go func() {
		for {
			if err := f.fetch(fallback); err != nil {
				log.Info().Err(err).Msg("fetch osquery config")
			}
			f.mu.Lock()
			done := !f.pending
			if done {
				f.fetching = false
			} else {
				f.pending = false
				fallback = false
			}
			f.mu.Unlock()
			f.notifyFetched()
			if done {
				return
			}
		}
	}()
}

func (f *ConfigFetcher) notifyFetched() {
	if f.fetched != nil {
		f.fetched()
	}
}

// fetch fetches the config from the server and caches it. fallback reports
// whether the fetch is a fallback poll.
func (f *ConfigFetcher) fetch(fallback bool) error {
	f.mu.Lock()
	etag, reportFallbackChange := f.etag, f.reportFallbackChange
	f.mu.Unlock()

	raw, err := f.client.OsqueryConfig(etag, reportFallbackChange)

	f.mu.Lock()
	defer f.mu.Unlock()
	if err != nil {
		f.scheduleNextLocked(false)
		return err
	}
	f.reportFallbackChange = false
	cfg, newETag, unchanged, err := prepareConfig(raw)
	if err != nil {
		f.scheduleNextLocked(false)
		return err
	}
	if unchanged {
		f.scheduleNextLocked(true)
		return nil
	}
	if fallback && f.config != "" && cfg != f.config {
		f.reportFallbackChange = true
		log.Debug().Msg("osquery config change found by fallback poll")
	}
	f.config, f.etag = cfg, newETag
	f.scheduleNextLocked(true)
	return nil
}

// scheduleNextLocked sets when the next fetch is due. A failed fetch is
// retried at the disconnected cadence, also while connected. f.mu must be
// held.
func (f *ConfigFetcher) scheduleNextLocked(ok bool) {
	interval := f.serverRefresh
	f.nextIsFallback = ok && f.connected()
	if f.nextIsFallback {
		interval = f.fallbackIntervalLocked()
	}
	f.nextFetch = f.now().Add(interval)
}

// fallbackIntervalLocked returns the fallback poll interval of a connected
// agent: never shorter than the disconnected one, so the WebSocket never adds
// server load. f.mu must be held.
func (f *ConfigFetcher) fallbackIntervalLocked() time.Duration {
	return max(f.serverRefresh, f.pollInterval)
}

// prepareConfig turns the server's config response into the config handed to
// osquery: the response without its etag. unchanged reports the server's
// response to a current etag.
func prepareConfig(raw json.RawMessage) (cfg, etag string, unchanged bool, err error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", "", false, fmt.Errorf("parse osquery config: %w", err)
	}
	if config == nil {
		return "", "", false, errors.New("parse osquery config: null")
	}
	if rawETag, ok := config["etag"]; ok {
		if err := json.Unmarshal(rawETag, &etag); err != nil {
			return "", "", false, fmt.Errorf("parse osquery config etag: %w", err)
		}
		delete(config, "etag")
	}
	if etag == configUnchangedETag {
		if len(config) > 0 {
			return "", "", false, errors.New("parse osquery config: unchanged response with a config")
		}
		return "", "", true, nil
	}
	b, err := json.Marshal(config)
	if err != nil {
		return "", "", false, fmt.Errorf("marshal osquery config: %w", err)
	}
	return string(b), etag, false, nil
}
