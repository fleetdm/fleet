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

// fakeExtensionManager simulates osquery rejecting registry items already
// registered by another extension.
type fakeExtensionManager struct {
	taken []string
	// alwaysConflict, if set, is reported as conflicting on every registration.
	alwaysConflict string

	mu         sync.Mutex
	registered [][]string
	pingErr    error
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
	return nil, errors.New("not implemented")
}

func (f *fakeExtensionManager) RegisterExtension(_ *osquerygen.InternalExtensionInfo, registry osquerygen.ExtensionRegistry) (*osquerygen.ExtensionStatus, error) {
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
	// Make Run return shortly after a successful registration.
	f.pingErr = errors.New("osquery gone")
	return &osquerygen.ExtensionStatus{Code: 0, UUID: 1}, nil
}

func (f *fakeExtensionManager) DeregisterExtension(osquerygen.ExtensionRouteUUID) (*osquerygen.ExtensionStatus, error) {
	return &osquerygen.ExtensionStatus{Code: 0}, nil
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

func TestRunnerSkipsConflictingTables(t *testing.T) {
	// Unix socket paths have a short length limit, so avoid t.TempDir().
	dir, err := os.MkdirTemp("", "ext")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s")

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
			plugins := []osquery.OsqueryPlugin{testTablePlugin("a"), testTablePlugin("b"), testTablePlugin("c")}
			r := NewRunner(sock)
			err := r.run(t.Context(), plugins, func() (*osquery.ExtensionManagerServer, error) {
				return osquery.NewExtensionManagerServer("test", sock,
					osquery.WithClient(tc.fake),
					osquery.ServerPingInterval(10*time.Millisecond))
			})
			require.ErrorContains(t, err, tc.wantErr)
			require.Equal(t, tc.wantRegistered, tc.fake.registered)
		})
	}
}
