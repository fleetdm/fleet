package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// AutoUpdateFleetMaintainedApps walks every active Fleet-maintained app
// installer and, where its pin state allows, downloads the newest published
// version into the team's cache and advances the active installer to it. When
// softwareInstallStore is nil (e.g. the installer store isn't configured) it
// degrades to promote-only: it advances among versions already cached but
// fetches nothing from upstream. A failure on one app is logged and skipped so a
// single bad row can't stall the whole run.
func AutoUpdateFleetMaintainedApps(ctx context.Context, ds fleet.Datastore, softwareInstallStore fleet.SoftwareInstallerStore, logger *slog.Logger) error {
	candidates, err := ds.ListFleetMaintainedAppActiveInstallers(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing active fleet-maintained app installers")
	}

	// One HTTP client and one manifest cache for the whole run, so each distinct
	// manifest is fetched at most once regardless of how many teams use the app.
	var client *http.Client
	if softwareInstallStore != nil {
		timeout := maintained_apps.InstallerTimeout
		if v := dev_mode.Env("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				timeout = d
			}
		}
		client = fleethttp.NewClient(fleethttp.WithTimeout(timeout))
	}
	manifests := map[string]*manifestEntry{}

	for _, c := range candidates {
		if err := autoUpdateOneFleetMaintainedApp(ctx, ds, softwareInstallStore, client, logger, c, manifests); err != nil {
			logger.ErrorContext(ctx, "auto-updating fleet-maintained app",
				"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug, "err", err)
		}
	}
	return nil
}

func autoUpdateOneFleetMaintainedApp(
	ctx context.Context,
	ds fleet.Datastore,
	store fleet.SoftwareInstallerStore,
	client *http.Client,
	logger *slog.Logger,
	c fleet.FMAAutoUpdateCandidate,
	manifests map[string]*manifestEntry,
) error {
	pinned, err := ds.GetPinnedVersion(ctx, c.TeamID, c.TitleID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ctxerr.Wrap(ctx, err, "getting pinned version")
	}
	pin := ""
	if pinned != nil {
		pin = *pinned
	}

	// A literal pin never advances and never pre-caches: the version is already
	// cached and nothing newer is wanted (re-pinning is handled on-demand).
	if pin != "" && !strings.HasPrefix(pin, "^") {
		return nil
	}

	// Download the latest published version when the store is configured and the
	// pin would actually promote to it. Download failures are isolated: we log and
	// still try to promote among whatever is already cached.
	if store != nil && client != nil {
		if err := downloadNewVersionIfEligible(ctx, ds, store, logger, c, pin, client, manifests); err != nil {
			logger.ErrorContext(ctx, "downloading new fleet-maintained app version",
				"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug, "err", err)
		}
	}

	return promoteFleetMaintainedApp(ctx, ds, logger, c, pin)
}

// promoteFleetMaintainedApp advances the active installer to the newest cached
// version the pin allows, without ever rewriting the pin (a nil PinnedVersion
// leaves it untouched, so an admin pin changed between this read and write is
// never clobbered).
func promoteFleetMaintainedApp(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, c fleet.FMAAutoUpdateCandidate, pin string) error {
	// Cached versions, semver-sorted newest-first.
	versions, err := ds.GetFleetMaintainedVersionsByTitleID(ctx, c.TeamID, c.TitleID, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting cached versions")
	}
	if len(versions) == 0 {
		return nil
	}

	target, ok := selectAutoUpdateTarget(versions, pin)
	if !ok || target.ID == c.InstallerID {
		// No newer eligible version, or already on it.
		return nil
	}

	payload := &fleet.UpdateSoftwareInstallerPayload{
		TeamID:  c.TeamID,
		TitleID: c.TitleID,
	}
	// SetFleetMaintainedAppActiveInstaller atomically flips the active installer,
	// re-points policies, and redirects installs frozen on the old version to the
	// new one, so a host never installs a version other than the one displayed.
	if err := ds.SetFleetMaintainedAppActiveInstaller(ctx, payload, target.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting active installer")
	}

	logger.InfoContext(ctx, "advanced fleet-maintained app to newer cached version",
		"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug,
		"from", c.Version, "to", target.Version, "pin", pin)
	return nil
}

// manifestEntry memoizes the hydrated latest manifest (or its fetch error) for a
// slug across the run, so N teams sharing an app fetch the manifest once.
type manifestEntry struct {
	app *fleet.MaintainedApp
	err error
}

// downloadNewVersionIfEligible fetches the latest manifest, and if the pin would
// promote to it and it isn't already cached, downloads the installer and caches
// it (inactive) for the team. Promotion happens separately, after this returns,
// so the bytes are always stored before the active flag flips.
func downloadNewVersionIfEligible(
	ctx context.Context,
	ds fleet.Datastore,
	store fleet.SoftwareInstallerStore,
	logger *slog.Logger,
	c fleet.FMAAutoUpdateCandidate,
	pin string,
	client *http.Client,
	manifests map[string]*manifestEntry,
) error {
	app, err := hydrateLatestManifest(ctx, ds, c, manifests)
	if err != nil {
		return err
	}

	// For a concrete manifest version the eligibility gates can run up front. For a
	// "latest" manifest the real version isn't known until the installer is
	// extracted, so the caret-major and cache checks are deferred until after that.
	isLatest := app.Version == "latest"
	if !isLatest {
		// Caret pin: caching a version the pin can never promote to wastes a slot.
		// Empty pin always takes latest; literal pins were filtered out by the caller.
		if pin != "" && !versionMatchesMajor(app.Version, strings.TrimPrefix(pin, "^")) {
			return nil
		}
		has, err := ds.HasFMAInstallerVersion(ctx, c.TeamID, c.FleetMaintainedAppID, app.Version)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking cached version")
		}
		if has {
			return nil
		}
	}

	// Byte dedup: when the expected hash is known and already in the store (another
	// team cached the same version), skip the HTTP download and reuse the bytes.
	storageID := app.SHA256
	needBytes := true
	if app.SHA256 != noCheckHash && !isLatest {
		exists, err := store.Exists(ctx, app.SHA256)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking installer store")
		}
		needBytes = !exists
	}

	version := app.Version
	filename := ""
	upgradeCode := app.UpgradeCode
	var packageIDs []string
	var tfr *fleet.TempFileReader
	if needBytes {
		tfr, filename, err = maintained_apps.DownloadInstaller(ctx, app.InstallerURL, client)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "downloading app installer")
		}
		defer tfr.Close()

		gotHash, err := file.SHA256FromTempFileReader(tfr)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "calculating SHA256 hash")
		}
		if app.SHA256 != noCheckHash {
			if gotHash != app.SHA256 {
				return ctxerr.New(ctx, "mismatch in maintained app SHA256 hash")
			}
		} else {
			storageID = gotHash
		}

		// Extract the concrete version (for "latest") and the package IDs / upgrade
		// code needed to substitute the uninstall script. Best-effort except when
		// it's the only way to resolve a "latest" version. Rewind unconditionally —
		// extraction consumes the reader and the bytes are stored below.
		meta, metaErr := file.ExtractInstallerMetadata(tfr)
		if err := tfr.Rewind(); err != nil {
			return ctxerr.Wrap(ctx, err, "resetting installer file reader")
		}
		if metaErr != nil {
			if isLatest {
				return ctxerr.Wrap(ctx, metaErr, "extracting installer metadata")
			}
			logger.WarnContext(ctx, "extracting fleet-maintained app installer metadata", "slug", c.Slug, "err", metaErr)
		} else {
			if isLatest {
				version = meta.Version
			}
			packageIDs = meta.PackageIDs
			if meta.UpgradeCode != "" {
				upgradeCode = meta.UpgradeCode
			}
		}
	} else {
		// Bytes already cached (possibly only on an inactive row after a rollback):
		// recover the package IDs and upgrade code from any installer with the same
		// content hash so the uninstall script can still be substituted.
		pids, ucode, err := ds.GetSoftwareInstallerMetadataByStorageID(ctx, storageID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "recovering cached installer metadata")
		}
		packageIDs = pids
		if ucode != "" {
			upgradeCode = ucode
		}
	}

	// Apply the deferred gates now that the concrete version is known ("latest").
	if isLatest && pin != "" && !versionMatchesMajor(version, strings.TrimPrefix(pin, "^")) {
		return nil
	}
	if version != app.Version {
		has, err := ds.HasFMAInstallerVersion(ctx, c.TeamID, c.FleetMaintainedAppID, version)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking cached version")
		}
		if has {
			return nil
		}
	}

	if filename == "" { // bytes were reused; derive a filename from the URL
		if u, err := url.Parse(app.InstallerURL); err == nil {
			filename = path.Base(u.Path)
		}
	}

	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:          c.TeamID,
		Version:         version,
		Filename:        filename,
		Extension:       strings.TrimLeft(filepath.Ext(filename), "."),
		StorageID:       storageID,
		URL:             app.InstallerURL,
		UpgradeCode:     upgradeCode,
		PatchQuery:      app.PatchQuery,
		InstallScript:   app.InstallScript,
		UninstallScript: app.UninstallScript,
		PackageIDs:      packageIDs,
		InstallerFile:   tfr,
	}

	// Preserve admin-customized scripts across auto-updates. The active installer
	// (still the previous version here; promotion happens later) is the one to
	// carry forward from. Detect customization per-script by comparing against the
	// manifest: the install script is a version-independent template (direct
	// compare), but the uninstall script is version-specific after $PACKAGE_ID /
	// $UPGRADE_CODE substitution, so compare against the manifest template
	// substituted with the active version's package IDs.
	active, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, c.TeamID, c.TitleID, true)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "getting active installer to preserve custom scripts")
	}
	if active != nil {
		if strings.TrimSpace(active.InstallScript) != strings.TrimSpace(app.InstallScript) {
			payload.InstallScript = active.InstallScript
		}
		defaultUninstall := &fleet.UploadSoftwareInstallerPayload{
			UninstallScript: app.UninstallScript,
			PackageIDs:      active.PackageIDs(),
			UpgradeCode:     active.UpgradeCode,
			Extension:       active.Extension,
		}
		if err := preProcessUninstallScript(defaultUninstall); err != nil {
			return ctxerr.Wrap(ctx, err, "computing manifest uninstall script for comparison")
		}
		if strings.TrimSpace(active.UninstallScript) != strings.TrimSpace(defaultUninstall.UninstallScript) {
			payload.UninstallScript = active.UninstallScript
		}
	}

	// Substitute $PACKAGE_ID / $UPGRADE_CODE in the uninstall script, matching the
	// GitOps materialization path (no-op when there are no package IDs).
	if err := preProcessUninstallScript(payload); err != nil {
		return ctxerr.Wrap(ctx, err, "processing uninstall script")
	}
	// Refuse to persist a row whose uninstall script still has unsubstituted
	// template variables (e.g. metadata extraction failed and preProcess silently
	// no-op'd): promoting it would record uninstalls as succeeding while the app
	// stays installed. Skip this candidate; the next run retries.
	if file.PackageIDRegex.MatchString(payload.UninstallScript) || file.UpgradeCodeRegex.MatchString(payload.UninstallScript) {
		return ctxerr.Errorf(ctx, "uninstall script for %q still has unsubstituted template variables; skipping cache", c.Slug)
	}

	// Store the bytes before creating the DB row, so a Put failure can't leave a
	// row pointing at installer bytes that aren't in the store — which the caller
	// would then promote the active installer to.
	if needBytes && tfr != nil {
		exists, err := store.Exists(ctx, storageID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking installer store")
		}
		if !exists {
			if err := store.Put(ctx, storageID, tfr); err != nil {
				return ctxerr.Wrap(ctx, err, "storing installer")
			}
		}
	}

	if _, err := ds.InsertFleetMaintainedAppVersion(ctx, c.InstallerID, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "caching new fleet-maintained app version")
	}

	logger.InfoContext(ctx, "cached new fleet-maintained app version",
		"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug,
		"from", c.Version, "to", version, "pin", pin)
	return nil
}

func hydrateLatestManifest(ctx context.Context, ds fleet.Datastore, c fleet.FMAAutoUpdateCandidate, manifests map[string]*manifestEntry) (*fleet.MaintainedApp, error) {
	if e, ok := manifests[c.Slug]; ok {
		return e.app, e.err
	}
	app, err := func() (*fleet.MaintainedApp, error) {
		skeleton, err := ds.GetMaintainedAppByID(ctx, c.FleetMaintainedAppID, c.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting maintained app by id")
		}
		// version "" fetches the latest from the remote manifest; teamID/cache are unused.
		hydrated, err := maintained_apps.Hydrate(ctx, skeleton, "", c.TeamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "hydrating app from manifest")
		}
		return hydrated, nil
	}()
	manifests[c.Slug] = &manifestEntry{app: app, err: err}
	return app, err
}

// selectAutoUpdateTarget picks the cached version the pin allows the cron to
// advance to. versions must be semver-sorted newest-first. An empty pin means
// Latest (newest). A caret pin returns the newest version within its major, or
// ok=false when no cached version satisfies the major (so the cron skips rather
// than crossing into another major, unlike the on-demand PATCH path).
func selectAutoUpdateTarget(versions []fleet.FleetMaintainedVersion, pin string) (fleet.FleetMaintainedVersion, bool) {
	if pin == "" {
		return versions[0], true
	}
	// Caret pin: parsePinnedVersion already validated the shape on write.
	major := strings.TrimPrefix(pin, "^")
	for _, v := range versions {
		if versionMatchesMajor(v.Version, major) {
			return v, true
		}
	}
	return fleet.FleetMaintainedVersion{}, false
}

// teamIDForLog renders an optional team ID for structured logs; slog prints a
// *uint as its address, so the value (or "none") is surfaced here instead.
func teamIDForLog(p *uint) any {
	if p == nil {
		return "none"
	}
	return *p
}
