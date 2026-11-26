//go:build !windows

// Excluded on Windows because the test listens on a unix socket path, which
// isn't a valid named pipe path.

package table

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/osquery/osquery-go"
	osquerygen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

// fakeExtensionManager simulates osquery rejecting registry items already
// registered by another extension.
type fakeExtensionManager struct {
	taken []string
	// alwaysConflict, if set, is reported as conflicting on every registration.
	alwaysConflict string
	// stayUp keeps pings succeeding after a successful registration.
	stayUp bool
	// extensionsErr, if set, is returned when listing extensions.
	extensionsErr error

	mu         sync.Mutex
	registered [][]string
	pingErr    error
	extensions osquerygen.InternalExtensionList
}

// evict simulates osquery dropping the registered extension ("Extension UUID N
// has gone away").
func (f *fakeExtensionManager) evict() {
	f.mu.Lock()
	defer f.mu.Unlock()
	clear(f.extensions)
}

func (f *fakeExtensionManager) Close() {}

func (f *fakeExtensionManager) Ping() (*osquerygen.ExtensionStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pingErr != nil {
		return nil, f.pingErr
	}
	return &osquerygen.ExtensionStatus{Code: 0}, nil
}

func (f *fakeExtensionManager) Call(string, string, osquerygen.ExtensionPluginRequest) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeExtensionManager) Extensions() (osquerygen.InternalExtensionList, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.extensionsErr != nil {
		return nil, f.extensionsErr
	}
	return maps.Clone(f.extensions), nil
}

func (f *fakeExtensionManager) RegisterExtension(info *osquerygen.InternalExtensionInfo, registry osquerygen.ExtensionRegistry) (*osquerygen.ExtensionStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var names []string
	for name := range registry["table"] {
		names = append(names, name)
	}
	slices.Sort(names)
	f.registered = append(f.registered, names)
	conflict := f.alwaysConflict
	for _, name := range names {
		if slices.Contains(f.taken, name) {
			conflict = name
			break
		}
	}
	if conflict != "" {
		return &osquerygen.ExtensionStatus{
			Code:    1,
			Message: "Failed adding registry: Registry item " + conflict + " conflicts with another item of the same name.",
		}, nil
	}
	f.extensions = osquerygen.InternalExtensionList{1: info}
	if !f.stayUp {
		// Make Run return shortly after a successful registration.
		f.pingErr = errors.New("osquery gone")
	}
	return &osquerygen.ExtensionStatus{Code: 0, UUID: 1}, nil
}

func (f *fakeExtensionManager) DeregisterExtension(osquerygen.ExtensionRouteUUID) (*osquerygen.ExtensionStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// With osquery.WithClient, Shutdown doesn't drop the client, so osquery-go's
	// ping goroutine would keep pinging forever; failing pings stops it.
	if f.pingErr == nil {
		f.pingErr = errors.New("deregistered")
	}
	return &osquerygen.ExtensionStatus{Code: 0}, nil
}

func (f *fakeExtensionManager) getRegistered() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.registered
}

func (f *fakeExtensionManager) Options() (osquerygen.InternalOptionList, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeExtensionManager) Query(string) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeExtensionManager) GetQueryColumns(string) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func testTablePlugin(name string) osquery.OsqueryPlugin {
	return table.NewPlugin(name, []table.ColumnDefinition{table.TextColumn("c")},
		func(context.Context, table.QueryContext) ([]map[string]string, error) { return nil, nil })
}

// newTestRunner returns a function that runs a Runner connected to f.
func newTestRunner(t *testing.T, f *fakeExtensionManager) func(ctx context.Context, plugins []osquery.OsqueryPlugin) error {
	// Unix socket paths have a short length limit, so avoid t.TempDir().
	// A dir per test avoids reusing a listen path that osquery-go may
	// still be closing asynchronously.
	dir, err := os.MkdirTemp("", "ext")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s")

	r := NewRunner(sock)
	// The runner returns without stopping the server when osquery drops the
	// extension, leaving it to Interrupt.
	t.Cleanup(func() {
		if srv := r.getSrv(); srv != nil {
			_ = srv.Shutdown(context.Background())
		}
	})
	r.checkInterval = 10 * time.Millisecond
	r.newClient = func(time.Duration) (osquery.ExtensionManager, error) { return f, nil }
	return func(ctx context.Context, plugins []osquery.OsqueryPlugin) error {
		return r.run(ctx, plugins, func(client osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error) {
			return osquery.NewExtensionManagerServer("test", sock,
				osquery.WithClient(client),
				osquery.ServerPingInterval(10*time.Millisecond))
		})
	}
}

func TestRunnerSkipsConflictingTables(t *testing.T) {
	for _, tc := range []struct {
		name           string
		fake           *fakeExtensionManager
		wantRegistered [][]string
		wantErr        string
	}{
		{
			name:           "no conflicts",
			fake:           &fakeExtensionManager{},
			wantRegistered: [][]string{{"a", "b", "c"}},
			wantErr:        "osquery gone",
		},
		{
			name:           "two conflicts",
			fake:           &fakeExtensionManager{taken: []string{"a", "c"}},
			wantRegistered: [][]string{{"a", "b", "c"}, {"b", "c"}, {"b"}},
			wantErr:        "osquery gone",
		},
		{
			name:           "conflict on item not owned by fleetd",
			fake:           &fakeExtensionManager{alwaysConflict: "z"},
			wantRegistered: [][]string{{"a", "b", "c"}},
			wantErr:        "Registry item z conflicts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := newTestRunner(t, tc.fake)
			plugins := []osquery.OsqueryPlugin{testTablePlugin("a"), testTablePlugin("b"), testTablePlugin("c")}
			err := run(t.Context(), plugins)
			require.ErrorContains(t, err, tc.wantErr)
			require.Equal(t, tc.wantRegistered, tc.fake.getRegistered())
		})
	}
}

func TestRunnerReturnsWhenOsqueryDropsExtension(t *testing.T) {
	f := &fakeExtensionManager{stayUp: true}
	run := newTestRunner(t, f)
	errc := make(chan error, 1)
	go func() {
		errc <- run(t.Context(), []osquery.OsqueryPlugin{testTablePlugin("a")})
	}()

	require.Eventually(t, func() bool { return len(f.getRegistered()) == 1 }, 5*time.Second, 10*time.Millisecond)
	// Still registered: several checks pass without ending the session.
	select {
	case err := <-errc:
		t.Fatalf("runner returned while registered: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	f.evict()
	select {
	case err := <-errc:
		require.ErrorIs(t, err, errExtensionNotRegistered)
	case <-time.After(5 * time.Second):
		t.Fatal("runner didn't notice osquery dropped the extension")
	}
}

func TestRunnerReturnsWhenRegistrationCantBeChecked(t *testing.T) {
	f := &fakeExtensionManager{stayUp: true}
	run := newTestRunner(t, f)
	errc := make(chan error, 1)
	go func() {
		errc <- run(t.Context(), []osquery.OsqueryPlugin{testTablePlugin("a")})
	}()

	require.Eventually(t, func() bool { return len(f.getRegistered()) == 1 }, 5*time.Second, 10*time.Millisecond)
	f.mu.Lock()
	f.extensionsErr = errors.New("connection refused")
	f.mu.Unlock()
	select {
	case err := <-errc:
		require.ErrorContains(t, err, "check extension registration: connection refused")
	case <-time.After(5 * time.Second):
		t.Fatal("runner didn't give up checking the registration")
	}
}
