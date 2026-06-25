package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setBootEnv sets an env var for the duration of the (sub)test, restoring the
// prior value on cleanup. We use os.Setenv rather than t.Setenv because the
// MySQL test helper marks the parent test as parallel, which t.Setenv disallows.
// The boot scenarios run as serial subtests so these process-global env vars do
// not race between scenarios.
func setBootEnv(t *testing.T, key, value string) {
	prev, had := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func freeLocalAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// fatalRecorder swaps the package-level initFatal var to collect calls instead
// of terminating the test binary, restoring the original on cleanup.
type fatalRecorder struct{ calls []string }

func installFatalRecorder(t *testing.T) *fatalRecorder {
	r := &fatalRecorder{}
	orig := initFatal
	initFatal = func(err error, msg string) {
		r.calls = append(r.calls, msg+": "+err.Error())
	}
	t.Cleanup(func() { initFatal = orig })
	return r
}

func (r *fatalRecorder) contains(substr string) bool {
	for _, c := range r.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// configureBootEnv points the server's config at the given migrated test
// database and the test Redis (on the given Redis logical database, so serial
// boot scenarios don't share cron locks/state), on a free local port, with TLS
// off. It returns the server's address.
func configureBootEnv(t *testing.T, dbName string, redisDB int) string {
	serverAddr := freeLocalAddr(t)
	redisAddr := os.Getenv("REDIS_TEST_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	setBootEnv(t, "FLEET_MYSQL_ADDRESS", testing_utils.TestAddress)
	setBootEnv(t, "FLEET_MYSQL_USERNAME", testing_utils.TestUsername)
	setBootEnv(t, "FLEET_MYSQL_PASSWORD", testing_utils.TestPassword)
	setBootEnv(t, "FLEET_MYSQL_DATABASE", dbName)
	setBootEnv(t, "FLEET_REDIS_ADDRESS", redisAddr)
	setBootEnv(t, "FLEET_REDIS_DATABASE", strconv.Itoa(redisDB))
	setBootEnv(t, "FLEET_SERVER_ADDRESS", serverAddr)
	setBootEnv(t, "FLEET_SERVER_TLS", "false")
	// The test schema is loaded from a dump that does not mark every data
	// migration as applied, so allow the server to boot past the migration
	// status check (the schema is functionally complete for a boot test).
	setBootEnv(t, "FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS", "1")

	return serverAddr
}

// runServe runs the serve command with a cancelable context and any extra
// command-line flags, returning a channel that receives the command's exit
// error when runServeCmd returns.
func runServe(ctx context.Context, extraArgs ...string) <-chan error {
	rootCmd := createRootCmd()
	configManager := config.NewManager(rootCmd)
	rootCmd.AddCommand(createServeCmd(configManager))
	rootCmd.SetArgs(append([]string{"serve", "--dev_license"}, extraArgs...))

	done := make(chan error, 1)
	go func() { done <- rootCmd.ExecuteContext(ctx) }()
	return done
}

func waitHealthy(t *testing.T, serverAddr string) bool {
	return assert.Eventually(t, func() bool {
		resp, err := http.Get("http://" + serverAddr + "/healthz") //nolint:gosec
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 60*time.Second, 250*time.Millisecond)
}

func waitShutdown(t *testing.T, done <-chan error) {
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(60 * time.Second):
		t.Fatal("server did not shut down within 60s of context cancellation")
	}
}

// TestRunServeCmd boots the full server via runServeCmd against a real (migrated)
// test MySQL and Redis and exercises the entire startup path end to end. The
// scenarios run as serial subtests sharing one database so the process-global
// boot env vars do not race.
func TestRunServeCmd(t *testing.T) {
	if os.Getenv("MYSQL_TEST") == "" || os.Getenv("REDIS_TEST") == "" {
		t.Skip("requires MYSQL_TEST=1 and REDIS_TEST=1")
	}

	const dbName = "fleet_serve_boot_test"
	mysqltest.CreateMySQLDSWithOptions(t, &testing_utils.DatastoreTestOptions{UniqueTestName: dbName})

	// Boots the full server and shuts it down gracefully on context
	// cancellation. A server private key is set so the boot also brings up the
	// Apple MDM protocol services and the host-identity / conditional-access
	// SCEP setup, exercising the MDM-enabled startup path.
	//
	// NOTE: runServeCmd registers metrics collectors with the process-global
	// Prometheus registry, which can only happen once per process, so this is
	// the single full boot in this package. Error-path scenarios below must fail
	// before that registration.
	t.Run("boots with Apple MDM enabled and shuts down gracefully", func(t *testing.T) {
		rec := installFatalRecorder(t)
		serverAddr := configureBootEnv(t, dbName, 12)
		setBootEnv(t, "FLEET_SERVER_PRIVATE_KEY", strings.Repeat("a", 32))

		ctx, cancel := context.WithCancel(context.Background())
		done := runServe(ctx)

		healthy := waitHealthy(t, serverAddr)
		require.Emptyf(t, rec.calls, "initFatal was called during boot: %v", rec.calls)
		require.True(t, healthy, "server did not become healthy")

		cancel()
		waitShutdown(t, done)
	})

	// An invalid Redis host-cache configuration (enabled with a non-positive
	// TTL) must make the server fail fast through initFatal and return rather
	// than start serving, exercising the Redis-init error path and the nil-pool
	// guard in runServeCmd.
	t.Run("refuses to boot on invalid host-cache config", func(t *testing.T) {
		rec := installFatalRecorder(t)
		configureBootEnv(t, dbName, 13)
		setBootEnv(t, "FLEET_REDIS_HOST_CACHE_ENABLED", "true")
		setBootEnv(t, "FLEET_REDIS_HOST_CACHE_TTL", "0")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := runServe(ctx)

		select {
		case <-done:
		case <-time.After(60 * time.Second):
			t.Fatal("server did not return after invalid host-cache config")
		}

		require.NotEmpty(t, rec.calls, "expected initFatal for invalid host-cache config")
		assert.Truef(t, rec.contains("host_cache_ttl must be > 0"),
			"expected a host-cache validation failure, got: %v", rec.calls)
	})
}
