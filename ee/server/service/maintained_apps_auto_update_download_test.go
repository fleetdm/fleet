package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	mocksoftware "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/stretchr/testify/require"
)

const (
	testFMASlug    = "chrome/darwin"
	testFMALatest  = "150.0.0"
	testFMAAppID   = uint(5)
	testFMATitleID = uint(1)
)

// fakeManifestServer serves the latest manifest for testFMASlug plus the
// installer it points at, and counts hits so tests can assert fetch-once and
// byte-dedup behavior.
type fakeManifestServer struct {
	srv           *httptest.Server
	sha           string
	bytes         []byte
	manifestHits  int
	installerHits int
	mu            sync.Mutex
}

func newFakeManifestServer(t *testing.T) *fakeManifestServer {
	f := &fakeManifestServer{bytes: []byte("fake installer payload")}
	sum := sha256.Sum256(f.bytes)
	f.sha = hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/"+testFMASlug+".json", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.manifestHits++
		f.mu.Unlock()
		manifest := ma.FMAManifestFile{
			Versions: []*ma.FMAManifestApp{{
				Version:            testFMALatest,
				InstallerURL:       f.srv.URL + "/installer.pkg",
				SHA256:             f.sha,
				InstallScriptRef:   "i",
				UninstallScriptRef: "u",
				Queries:            ma.FMAQueries{Exists: "SELECT 1", Patched: "SELECT 2"},
				DefaultCategories:  []string{"Browsers"},
			}},
			Refs: map[string]string{"i": "echo install", "u": "echo uninstall"},
		}
		_ = json.NewEncoder(w).Encode(manifest)
	})
	mux.HandleFunc("/installer.pkg", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.installerHits++
		f.mu.Unlock()
		_, _ = w.Write(f.bytes)
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", f.srv.URL, t)
	return f
}

// memStore is a stateful in-memory SoftwareInstallerStore so byte-dedup across
// teams behaves like the real store (a Put makes a later Exists true).
func memStore(seed ...string) *mocksoftware.SoftwareInstallerStore {
	var mu sync.Mutex
	have := map[string]struct{}{}
	for _, s := range seed {
		have[s] = struct{}{}
	}
	store := &mocksoftware.SoftwareInstallerStore{}
	store.ExistsFunc = func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		_, ok := have[id]
		return ok, nil
	}
	store.PutFunc = func(ctx context.Context, id string, content io.ReadSeeker) error {
		mu.Lock()
		have[id] = struct{}{}
		mu.Unlock()
		return nil
	}
	return store
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// baseDownloadStore wires a mock datastore for the download-then-promote flow:
// one unpinned candidate on the given version, hydrating from the fake server.
func baseDownloadStore(t *testing.T, activeVersion string, activeID uint) *mock.Store {
	ds := new(mock.Store)
	teamID := uint(1)
	ds.ListFleetMaintainedAppActiveInstallersFunc = func(ctx context.Context) ([]fleet.FMAAutoUpdateCandidate, error) {
		return []fleet.FMAAutoUpdateCandidate{{
			TeamID: &teamID, TitleID: testFMATitleID, FleetMaintainedAppID: testFMAAppID,
			InstallerID: activeID, Version: activeVersion, Slug: testFMASlug,
		}}, nil
	}
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) {
		return nil, nil // Latest
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, tmID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{ID: testFMAAppID, Name: "Google Chrome", Slug: testFMASlug, Platform: "darwin"}, nil
	}
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, error) {
		return false, nil
	}
	// After the insert, the new version is the newest cached one.
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: testFMALatest}, {ID: activeID, Version: activeVersion}}, nil
	}
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		require.Nil(t, payload.PinnedVersion, "cron must not write the pin row")
		return nil
	}
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, a, b bool) error { return nil }
	return ds
}

func TestAutoUpdateDownloadsAndPromotes(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)

	var gotActiveInstaller uint
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotActiveInstaller = activeInstallerID
		gotPayload = payload
		return 13, nil
	}

	store := memStore()
	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, store, discardLogger()))

	// Downloaded, validated, and cached the new version.
	require.Equal(t, 1, srv.installerHits)
	require.True(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
	require.NotNil(t, gotPayload)
	require.Equal(t, uint(9), gotActiveInstaller, "clones from the current active installer")
	require.Equal(t, testFMALatest, gotPayload.Version)
	require.Equal(t, srv.sha, gotPayload.StorageID)
	require.Equal(t, "installer.pkg", gotPayload.Filename)
	require.Equal(t, "pkg", gotPayload.Extension)
	require.Equal(t, "echo install", gotPayload.InstallScript)
	require.True(t, store.PutFuncInvoked, "stores bytes before promotion")

	// Then promoted to the freshly cached version.
	require.True(t, ds.SetFleetMaintainedAppActiveInstallerFuncInvoked)
}

func TestAutoUpdateByteDedupSkipsDownload(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		require.Equal(t, "installer.pkg", payload.Filename, "filename derived from URL when bytes reused")
		return 13, nil
	}

	store := memStore(srv.sha) // bytes already present (another team cached them)
	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, store, discardLogger()))

	require.Equal(t, 0, srv.installerHits, "must not re-download bytes already in the store")
	require.True(t, ds.InsertFleetMaintainedAppVersionFuncInvoked, "still creates the per-team row")
	require.False(t, store.PutFuncInvoked)
}

func TestAutoUpdateAlreadyCachedSkipsInsert(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, error) {
		return true, nil // already cached
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("must not insert when the version is already cached")
		return 0, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.Equal(t, 0, srv.installerHits)
	require.False(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
	// Promotion among cached still runs.
	require.True(t, ds.GetFleetMaintainedVersionsByTitleIDFuncInvoked)
}

func TestAutoUpdateCaretMajorExceededSkipsDownload(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "147.0.5", 8)
	pin := "^147" // latest is 150.x — out of the pinned major
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) {
		return &pin, nil
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("must not download/cache a version outside the pinned major")
		return 0, nil
	}
	// Only an in-major version is cached; promotion stays within the major.
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 8, Version: "147.0.5"}}, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.Equal(t, 1, srv.manifestHits, "manifest fetched to learn the latest version")
	require.Equal(t, 0, srv.installerHits, "no download outside the pinned major")
	require.False(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
}

func TestAutoUpdateFetchesManifestOncePerSlug(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := new(mock.Store)
	teamA, teamB := uint(1), uint(2)
	ds.ListFleetMaintainedAppActiveInstallersFunc = func(ctx context.Context) ([]fleet.FMAAutoUpdateCandidate, error) {
		return []fleet.FMAAutoUpdateCandidate{
			{TeamID: &teamA, TitleID: testFMATitleID, FleetMaintainedAppID: testFMAAppID, InstallerID: 9, Version: "149.0.0", Slug: testFMASlug},
			{TeamID: &teamB, TitleID: 2, FleetMaintainedAppID: testFMAAppID, InstallerID: 19, Version: "149.0.0", Slug: testFMASlug},
		}, nil
	}
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) { return nil, nil }
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, tmID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{ID: testFMAAppID, Name: "Google Chrome", Slug: testFMASlug, Platform: "darwin"}, nil
	}
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, error) { return false, nil }
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		return 13, nil
	}
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: testFMALatest}}, nil
	}
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error { return nil }
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, a, b bool) error { return nil }

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))

	require.Equal(t, 1, srv.manifestHits, "manifest fetched once per slug across teams")
	require.Equal(t, 1, srv.installerHits, "bytes downloaded once, reused across teams via the store")
}
