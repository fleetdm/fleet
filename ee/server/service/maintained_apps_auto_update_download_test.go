package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	version       string // manifest version to advertise (default testFMALatest)
	uninstall     string // uninstall script ref body (default "echo uninstall")
	upgradeCode   string // manifest upgrade_code (default empty)
	manifestHits  int
	installerHits int
	mu            sync.Mutex
}

func newFakeManifestServer(t *testing.T) *fakeManifestServer {
	f := &fakeManifestServer{bytes: []byte("fake installer payload"), version: testFMALatest, uninstall: "echo uninstall"}
	sum := sha256.Sum256(f.bytes)
	f.sha = hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/"+testFMASlug+".json", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.manifestHits++
		f.mu.Unlock()
		manifest := ma.FMAManifestFile{
			Versions: []*ma.FMAManifestApp{{
				Version:            f.version,
				InstallerURL:       f.srv.URL + "/installer.pkg",
				SHA256:             f.sha,
				UpgradeCode:        f.upgradeCode,
				InstallScriptRef:   "i",
				UninstallScriptRef: "u",
				Queries:            ma.FMAQueries{Exists: "SELECT 1", Patched: "SELECT 2"},
				DefaultCategories:  []string{"Browsers"},
			}},
			Refs: map[string]string{"i": "echo install", "u": f.uninstall},
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
	// No recoverable metadata by default (byte-dedup path).
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) ([]string, string, error) {
		return nil, "", nil
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
	// By default the active installer has no custom scripts to carry forward, so the
	// cron keeps the manifest scripts. nil signals "nothing to preserve". Tests that
	// exercise custom-script carry-forward override this.
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}
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
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) ([]string, string, error) {
		return nil, "", nil
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		return 13, nil
	}
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: testFMALatest}}, nil
	}
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		return nil
	}
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, a, b bool) error { return nil }
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))

	require.Equal(t, 1, srv.manifestHits, "manifest fetched once per slug across teams")
	require.Equal(t, 1, srv.installerHits, "bytes downloaded once, reused across teams via the store")
}

// [5] A store.Put failure must NOT leave a DB row (which the caller would then
// promote to byte-less storage). Bytes are stored before the row is inserted.
func TestAutoUpdatePutFailureSkipsInsert(t *testing.T) {
	_ = newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("must not insert the DB row when storing bytes fails")
		return 0, nil
	}
	store := &mocksoftware.SoftwareInstallerStore{}
	store.ExistsFunc = func(ctx context.Context, id string) (bool, error) { return false, nil }
	store.PutFunc = func(ctx context.Context, id string, content io.ReadSeeker) error {
		return errors.New("store unavailable")
	}

	// The candidate's download errors, but the run is isolated and returns nil.
	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, store, discardLogger()))
	require.False(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
}

// [4] The uninstall script's $PACKAGE_ID is substituted (here via the byte-dedup
// path, where package IDs are recovered from the existing same-content installer).
func TestAutoUpdateSubstitutesUninstallScript(t *testing.T) {
	srv := newFakeManifestServer(t)
	srv.uninstall = "msiexec /x $PACKAGE_ID /qn"
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) ([]string, string, error) {
		return []string{"ABC"}, "", nil
	}
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotPayload = payload
		return 13, nil
	}

	store := memStore(srv.sha) // byte-dedup: no download, package IDs come from the lookup
	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, store, discardLogger()))
	require.NotNil(t, gotPayload)
	require.NotContains(t, gotPayload.UninstallScript, "$PACKAGE_ID", "placeholder must be substituted")
	require.Contains(t, gotPayload.UninstallScript, "ABC")
}

// [6] A caret pin with a "latest" manifest must not early-return before the real
// version is resolved — it should proceed to download (then bail here because the
// fake bytes can't be parsed).
func TestAutoUpdateCaretLatestAttemptsDownload(t *testing.T) {
	srv := newFakeManifestServer(t)
	srv.version = "latest"
	ds := baseDownloadStore(t, "150.0.0", 9)
	pin := "^150"
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) { return &pin, nil }
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("fake bytes can't resolve a latest version; insert should not happen")
		return 0, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.Equal(t, 1, srv.installerHits, "caret+latest must attempt the download, not early-return")
}

// [comment 1] When no package IDs can be recovered (e.g. metadata extraction
// fails) and the uninstall script still contains template variables, the cron
// must NOT persist/promote the version — otherwise uninstalls record success
// while the app stays installed.
func TestAutoUpdateUnsubstitutedUninstallSkipsInsert(t *testing.T) {
	srv := newFakeManifestServer(t)
	srv.uninstall = "msiexec /x $PACKAGE_ID /qn"
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("must not cache a version whose uninstall script still has $PACKAGE_ID")
		return 0, nil
	}

	// Download path: the fake bytes can't be parsed, so no package IDs are recovered
	// and the $PACKAGE_ID placeholder survives — the candidate must be skipped.
	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.False(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
}

// When the active installer has admin-customized scripts (differ from the
// manifest defaults), the cron carries them forward to the newly downloaded
// version instead of reverting to the manifest scripts.
func TestAutoUpdatePreservesCustomScripts(t *testing.T) {
	newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallScript:   "echo CUSTOM install",
			UninstallScript: "echo CUSTOM uninstall",
			Extension:       "pkg",
		}, nil
	}
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotPayload = payload
		return 13, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.NotNil(t, gotPayload)
	require.Equal(t, "echo CUSTOM install", gotPayload.InstallScript, "custom install script carried forward")
	require.Equal(t, "echo CUSTOM uninstall", gotPayload.UninstallScript, "custom uninstall script carried forward")
}
