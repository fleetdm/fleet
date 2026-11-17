//go:build !windows

// Excluded on Windows because the test listens on a unix socket path, which
// isn't a valid named pipe path.

package table

import (
	"context"
	"errors"
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

// fakeOsquery simulates osqueryd's extension manager.
type fakeOsquery struct {
	// taken are table names already registered by another extension.
	taken []string
	// alwaysConflict, if set, is reported as conflicting on every registration.
	alwaysConflict string
	// goneAfterRegister makes every ping fail after a successful registration.
	goneAfterRegister bool

	mu         sync.Mutex
	registered [][]string
	extensions map[osquerygen.ExtensionRouteUUID]string
	nextUUID   osquerygen.ExtensionRouteUUID
	gone       bool
	// failPings is how many pings fail before pings succeed again.
	failPings int
	// failDeregistrations is how many deregistrations fail.
	failDeregistrations int
}

// client returns a new connection to the fake osquery.
func (f *fakeOsquery) client() *fakeClient {
	return &fakeClient{f: f}
}

func (f *fakeOsquery) getRegistered() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.registered)
}

func (f *fakeOsquery) getExtensions() []osquerygen.ExtensionRouteUUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	var uuids []osquerygen.ExtensionRouteUUID
	for uuid := range f.extensions {
		uuids = append(uuids, uuid)
	}
	return uuids
}

// evictAll simulates osquery dropping registered extensions ("Extension UUID N
// has gone away").
func (f *fakeOsquery) evictAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	clear(f.extensions)
}

type fakeClient struct {
	f *fakeOsquery

	mu     sync.Mutex
	closed bool
}

func (c *fakeClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

func (c *fakeClient) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *fakeClient) Ping() (*osquerygen.ExtensionStatus, error) {
	if c.isClosed() {
		return nil, errors.New("client closed")
	}
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gone {
		return nil, errors.New("osquery gone")
	}
	if f.failPings > 0 {
		f.failPings--
		return nil, errors.New("broken pipe")
	}
	return &osquerygen.ExtensionStatus{Code: 0}, nil
}

func (c *fakeClient) Call(string, string, osquerygen.ExtensionPluginRequest) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClient) Extensions() (osquerygen.InternalExtensionList, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	list := osquerygen.InternalExtensionList{}
	for uuid, name := range f.extensions {
		list[uuid] = &osquerygen.InternalExtensionInfo{Name: name}
	}
	return list, nil
}

func (c *fakeClient) RegisterExtension(info *osquerygen.InternalExtensionInfo, registry osquerygen.ExtensionRegistry) (*osquerygen.ExtensionStatus, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	var names []string
	for name := range registry["table"] {
		names = append(names, name)
	}
	slices.Sort(names)
	f.registered = append(f.registered, names)
	for _, name := range f.extensions {
		if name == info.Name {
			return &osquerygen.ExtensionStatus{Code: 1, Message: "Duplicate extension registered"}, nil
		}
	}
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
	if f.goneAfterRegister {
		f.gone = true
	}
	if f.extensions == nil {
		f.extensions = make(map[osquerygen.ExtensionRouteUUID]string)
	}
	f.nextUUID++
	f.extensions[f.nextUUID] = info.Name
	return &osquerygen.ExtensionStatus{Code: 0, UUID: f.nextUUID}, nil
}

func (c *fakeClient) DeregisterExtension(uuid osquerygen.ExtensionRouteUUID) (*osquerygen.ExtensionStatus, error) {
	f := c.f
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failDeregistrations > 0 {
		f.failDeregistrations--
		return nil, errors.New("broken pipe")
	}
	delete(f.extensions, uuid)
	return &osquerygen.ExtensionStatus{Code: 0}, nil
}

func (c *fakeClient) Options() (osquerygen.InternalOptionList, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClient) Query(string) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClient) GetQueryColumns(string) (*osquerygen.ExtensionResponse, error) {
	return nil, errors.New("not implemented")
}

func testTablePlugin(name string) osquery.OsqueryPlugin {
	return table.NewPlugin(name, []table.ColumnDefinition{table.TextColumn("c")},
		func(context.Context, table.QueryContext) ([]map[string]string, error) { return nil, nil })
}

// newTestRunner returns a runner connected to f and a function to run it.
func newTestRunner(t *testing.T, f *fakeOsquery) (*Runner, func(ctx context.Context, plugins []osquery.OsqueryPlugin) error) {
	// Unix socket paths have a short length limit, so avoid t.TempDir().
	// A dir per test avoids reusing a listen path that osquery-go may
	// still be closing asynchronously.
	dir, err := os.MkdirTemp("", "ext")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s")

	r := NewRunner(sock)
	r.checkInterval = 10 * time.Millisecond
	r.retryBackoff = time.Millisecond
	r.newClient = func(time.Duration) (osquery.ExtensionManager, error) { return f.client(), nil }
	return r, func(ctx context.Context, plugins []osquery.OsqueryPlugin) error {
		return r.run(ctx, plugins, func(client osquery.ExtensionManager) (*osquery.ExtensionManagerServer, error) {
			return osquery.NewExtensionManagerServer("test", sock,
				osquery.WithClient(client),
				osquery.ServerPingInterval(10*time.Millisecond))
		})
	}
}

func repeat(names []string, n int) [][]string {
	var out [][]string
	for range n {
		out = append(out, names)
	}
	return out
}

func TestRunnerSkipsConflictingTables(t *testing.T) {
	// Each session fails right after registering, so the runner retries until it
	// gives up.
	attempts := maxConsecutiveSessionFailures + 1
	for _, tc := range []struct {
		name           string
		fake           *fakeOsquery
		wantRegistered [][]string
		wantErr        string
	}{
		{
			name:           "no conflicts",
			fake:           &fakeOsquery{goneAfterRegister: true},
			wantRegistered: repeat([]string{"a", "b", "c"}, attempts),
			wantErr:        "osquery gone",
		},
		{
			name: "two conflicts",
			fake: &fakeOsquery{taken: []string{"a", "c"}, goneAfterRegister: true},
			wantRegistered: append([][]string{{"a", "b", "c"}, {"b", "c"}},
				repeat([]string{"b"}, attempts)...),
			wantErr: "osquery gone",
		},
		{
			name:           "conflict on item not owned by fleetd",
			fake:           &fakeOsquery{alwaysConflict: "z"},
			wantRegistered: [][]string{{"a", "b", "c"}},
			wantErr:        "Registry item z conflicts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, run := newTestRunner(t, tc.fake)
			plugins := []osquery.OsqueryPlugin{testTablePlugin("a"), testTablePlugin("b"), testTablePlugin("c")}
			err := run(t.Context(), plugins)
			require.ErrorContains(t, err, tc.wantErr)
			require.Equal(t, tc.wantRegistered, tc.fake.getRegistered())
		})
	}
}

func TestRunnerReregisters(t *testing.T) {
	for _, tc := range []struct {
		name string
		fake *fakeOsquery
		// lose ends the first session.
		lose func(f *fakeOsquery)
	}{
		{
			name: "osquery evicts the extension",
			fake: &fakeOsquery{},
			lose: (*fakeOsquery).evictAll,
		},
		{
			name: "ping fails and deregistration fails",
			// Without deregistering the stale registration, osquery would reject
			// the new one as a duplicate.
			fake: &fakeOsquery{failDeregistrations: 1},
			lose: func(f *fakeOsquery) {
				f.mu.Lock()
				defer f.mu.Unlock()
				f.failPings = 1
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, run := newTestRunner(t, tc.fake)
			ctx, cancel := context.WithCancel(t.Context())
			errc := make(chan error, 1)
			go func() {
				errc <- run(ctx, []osquery.OsqueryPlugin{testTablePlugin("a")})
			}()

			require.Eventually(t, func() bool { return len(tc.fake.getExtensions()) == 1 }, 5*time.Second, 10*time.Millisecond)
			first := tc.fake.getExtensions()[0]
			tc.lose(tc.fake)

			require.Eventually(t, func() bool {
				uuids := tc.fake.getExtensions()
				return len(uuids) == 1 && uuids[0] != first
			}, 5*time.Second, 10*time.Millisecond)
			require.Equal(t, repeat([]string{"a"}, 2), tc.fake.getRegistered())

			cancel()
			select {
			case <-errc:
			case <-time.After(5 * time.Second):
				t.Fatal("runner didn't stop after cancel")
			}
			require.Empty(t, tc.fake.getExtensions())
		})
	}
}
