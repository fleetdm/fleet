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
	install       string // install script ref body (default "echo install")
	uninstall     string // uninstall script ref body (default "echo uninstall")
	upgradeCode   string // manifest upgrade_code (default empty)
	installerPath string // path the installer is served from (default "/installer.pkg")
	manifestHits  int
	installerHits int
	mu            sync.Mutex
}

func newFakeManifestServer(t *testing.T) *fakeManifestServer {
	return newFakeManifestServerWithInstaller(t, "/installer.pkg")
}

func newFakeManifestServerWithInstaller(t *testing.T, installerPath string) *fakeManifestServer {
	f := &fakeManifestServer{bytes: []byte("fake installer payload"), version: testFMALatest, install: "echo install", uninstall: "echo uninstall", installerPath: installerPath}
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
				InstallerURL:       f.srv.URL + f.installerPath,
				SHA256:             f.sha,
				UpgradeCode:        f.upgradeCode,
				InstallScriptRef:   "i",
				UninstallScriptRef: "u",
				Queries:            ma.FMAQueries{Exists: "SELECT 1", Patched: "SELECT 2"},
				DefaultCategories:  []string{"Browsers"},
			}},
			Refs: map[string]string{"i": f.install, "u": f.uninstall},
		}
		_ = json.NewEncoder(w).Encode(manifest)
	})
	mux.HandleFunc(installerPath, func(w http.ResponseWriter, r *http.Request) {
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
	return baseDownloadStoreWithEditedScripts(t, activeVersion, activeID, false, false)
}

func baseDownloadStoreWithEditedScripts(t *testing.T, activeVersion string, activeID uint, installEdited bool, uninstallEdited bool) *mock.Store {
	ds := new(mock.Store)
	teamID := uint(1)
	ds.ListFleetMaintainedAppActiveInstallersFunc = func(ctx context.Context) ([]fleet.FMAAutoUpdateCandidate, error) {
		return []fleet.FMAAutoUpdateCandidate{{
			TeamID: &teamID, TitleID: testFMATitleID, FleetMaintainedAppID: testFMAAppID,
			InstallerID: activeID, Version: activeVersion, Slug: testFMASlug,
			InstallScriptEdited: installEdited, UninstallScriptEdited: uninstallEdited,
		}}, nil
	}
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) {
		return nil, nil // Latest
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, tmID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{ID: testFMAAppID, Name: "Google Chrome", Slug: testFMASlug, Platform: "darwin"}, nil
	}
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, string, error) {
		return false, "", nil
	}
	// No recoverable metadata by default (byte-dedup path).
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) (fleet.CachedInstallerMetadata, error) {
		return fleet.CachedInstallerMetadata{}, nil
	}
	// After the insert, the new version is the newest cached one.
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: testFMALatest}, {ID: activeID, Version: activeVersion}}, nil
	}
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		require.Nil(t, payload.PinnedVersion, "cron must not write the pin row")
		return nil
	}
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, a, b bool) error { return nil }
	ds.MarkFleetMaintainedAppVersionCurrentFunc = func(ctx context.Context, installerID uint) error {
		return nil
	}
	// By default the active installer has no custom scripts to carry forward, so the
	// cron keeps the manifest scripts. nil signals "nothing to preserve". Tests that
	// exercise custom-script carry-forward override this.
	ds.GetSoftwareInstallerMetadataByTeamTitleAndInstallerIDFunc = func(ctx context.Context, tmID *uint, titleID uint, installerID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}
	return ds
}

func TestAutoUpdateDownloadsAndPromotes(t *testing.T) {
	srv := newFakeManifestServerWithInstaller(t, "/installer.PKG")
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
	require.Equal(t, "installer.PKG", gotPayload.Filename, "filename keeps the original casing")
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
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, string, error) {
		return true, srv.sha, nil // cached with the manifest's bytes
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		t.Fatal("must not insert when the version is already cached")
		return 0, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.Equal(t, 0, srv.installerHits)
	require.False(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
	require.False(t, ds.MarkFleetMaintainedAppVersionCurrentFuncInvoked,
		"the published version is already the newest download, so nothing is reordered")
	// Promotion among cached still runs.
	require.True(t, ds.GetFleetMaintainedVersionsByTitleIDFuncInvoked)
}

func TestAutoUpdateRebuiltCachedVersionRefreshes(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	// The manifest's version is cached, but under bytes Fleet no longer serves, so it is
	// downloaded again and the cached row is refreshed rather than left alone.
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, string, error) {
		return true, "stale-hash", nil
	}
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotPayload = payload
		return 7, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.Equal(t, 1, srv.installerHits, "downloads the rebuilt package")
	require.True(t, ds.InsertFleetMaintainedAppVersionFuncInvoked)
	require.NotNil(t, gotPayload)
	require.Equal(t, testFMALatest, gotPayload.Version, "same version")
	require.Equal(t, srv.sha, gotPayload.StorageID, "new bytes")
	require.False(t, ds.MarkFleetMaintainedAppVersionCurrentFuncInvoked,
		"the refresh already moved it to the front")
}

func TestAutoUpdateNoCheckHashMarksCachedVersionCurrent(t *testing.T) {
	srv := newFakeManifestServer(t)
	// Homebrew's no_check sentinel: no hash to compare before downloading.
	srv.sha = noCheckHash
	ds := baseDownloadStore(t, "149.0.0", 9)
	// A newer version was downloaded after the one the manifest publishes now, which is the
	// state a rollback leaves behind.
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: "151.0.0"}, {ID: 7, Version: testFMALatest}}, nil
	}
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, string, error) {
		return true, "some-hash", nil
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		return 7, nil
	}
	var markedInstallerID uint
	ds.MarkFleetMaintainedAppVersionCurrentFunc = func(ctx context.Context, installerID uint) error {
		markedInstallerID = installerID
		return nil
	}
	// The list above is returned unchanged after the mark, the way a lagging replica would.
	var activatedInstallerID uint
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		activatedInstallerID = activeInstallerID
		return nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	// Without a hash the bytes may turn out to be ones Fleet already had, so the version the
	// manifest publishes still has to become the newest download.
	require.Equal(t, uint(7), markedInstallerID)
	require.Equal(t, uint(7), activatedInstallerID)
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
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint) ([]fleet.FleetMaintainedVersion, error) {
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
	ds.HasFMAInstallerVersionFunc = func(ctx context.Context, tmID *uint, fmaID uint, version string) (bool, string, error) {
		return false, "", nil
	}
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) (fleet.CachedInstallerMetadata, error) {
		return fleet.CachedInstallerMetadata{}, nil
	}
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		return 13, nil
	}
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 13, Version: testFMALatest}}, nil
	}
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		return nil
	}
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, a, b bool) error { return nil }
	ds.MarkFleetMaintainedAppVersionCurrentFunc = func(ctx context.Context, installerID uint) error {
		return nil
	}
	ds.GetSoftwareInstallerMetadataByTeamTitleAndInstallerIDFunc = func(ctx context.Context, tmID *uint, titleID uint, installerID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
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
	ds.GetSoftwareInstallerMetadataByStorageIDFunc = func(ctx context.Context, storageID string) (fleet.CachedInstallerMetadata, error) {
		return fleet.CachedInstallerMetadata{PackageIDs: []string{"ABC"}, Filename: "cached-installer.msi", Extension: "msi"}, nil
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
	// The file was never downloaded, so these come off the same-content row.
	require.Equal(t, "cached-installer.msi", gotPayload.Filename)
	require.Equal(t, "msi", gotPayload.Extension)
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

func TestAutoUpdatePreservesEditedScripts(t *testing.T) {
	newFakeManifestServer(t)
	ds := baseDownloadStoreWithEditedScripts(t, "149.0.0", 9, true, true)
	ds.GetSoftwareInstallerMetadataByTeamTitleAndInstallerIDFunc = func(ctx context.Context, tmID *uint, titleID uint, installerID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallScript:         "echo CUSTOM install",
			UninstallScript:       "echo CUSTOM uninstall",
			Extension:             "pkg",
			InstallScriptEdited:   true,
			UninstallScriptEdited: true,
		}, nil
	}
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotPayload = payload
		return 13, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.NotNil(t, gotPayload)
	require.Equal(t, "echo CUSTOM install", gotPayload.InstallScript, "edited install script carried forward")
	require.Equal(t, "echo CUSTOM uninstall", gotPayload.UninstallScript, "edited uninstall script carried forward")
	require.True(t, gotPayload.InstallScriptEdited)
	require.True(t, gotPayload.UninstallScriptEdited)
}

func TestAutoUpdateAdoptsManifestScriptsWhenNotEdited(t *testing.T) {
	srv := newFakeManifestServer(t)
	ds := baseDownloadStore(t, "149.0.0", 9)
	// The active installer is always read, but neither flag is set on it, so its
	// scripts lose to the manifest.
	ds.GetSoftwareInstallerMetadataByTeamTitleAndInstallerIDFunc = func(ctx context.Context, tmID *uint, titleID uint, installerID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallScript:   "echo STALE install",
			UninstallScript: "echo STALE uninstall",
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
	require.Equal(t, srv.install, gotPayload.InstallScript)
	require.False(t, gotPayload.InstallScriptEdited)
	require.False(t, gotPayload.UninstallScriptEdited)
}

func TestAutoUpdatePicksUpEditsMadeDuringDownload(t *testing.T) {
	newFakeManifestServer(t)
	// The candidate says unedited and the active installer says otherwise, which is an
	// admin editing the script while the installer downloaded.
	ds := baseDownloadStore(t, "149.0.0", 9)
	ds.GetSoftwareInstallerMetadataByTeamTitleAndInstallerIDFunc = func(ctx context.Context, tmID *uint, titleID uint, installerID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallScript:       "echo JUST EDITED",
			UninstallScript:     "echo uninstall",
			Extension:           "pkg",
			InstallScriptEdited: true,
		}, nil
	}
	var gotPayload *fleet.UploadSoftwareInstallerPayload
	ds.InsertFleetMaintainedAppVersionFunc = func(ctx context.Context, activeInstallerID uint, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
		gotPayload = payload
		return 13, nil
	}

	require.NoError(t, AutoUpdateFleetMaintainedApps(context.Background(), ds, memStore(), discardLogger()))
	require.NotNil(t, gotPayload)
	require.Equal(t, "echo JUST EDITED", gotPayload.InstallScript, "an edit made during the download is carried forward")
	require.True(t, gotPayload.InstallScriptEdited)
	require.False(t, gotPayload.UninstallScriptEdited)
}
