package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// AutoUpdateFleetMaintainedApps walks every active Fleet-maintained app
// installer and, where its pin state allows, advances it to a newer version
// already cached for the team — it never fetches new versions from upstream.
// A failure on one app is logged and skipped so a single bad row can't stall
// the whole run.
func AutoUpdateFleetMaintainedApps(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) error {
	candidates, err := ds.ListFleetMaintainedAppActiveInstallers(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing active fleet-maintained app installers")
	}

	for _, c := range candidates {
		if err := autoUpdateOneFleetMaintainedApp(ctx, ds, logger, c); err != nil {
			logger.ErrorContext(ctx, "auto-updating fleet-maintained app",
				"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug, "err", err)
		}
	}
	return nil
}

func autoUpdateOneFleetMaintainedApp(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, c fleet.FMAAutoUpdateCandidate) error {
	pinned, err := ds.GetPinnedVersion(ctx, c.TeamID, c.TitleID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ctxerr.Wrap(ctx, err, "getting pinned version")
	}
	pin := ""
	if pinned != nil {
		pin = *pinned
	}

	// A literal pin never advances.
	if pin != "" && !strings.HasPrefix(pin, "^") {
		return nil
	}

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
		TeamID:        c.TeamID,
		TitleID:       c.TitleID,
		PinnedVersion: &pin,
	}
	if err := ds.SetFleetMaintainedAppActiveInstaller(ctx, payload, target.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting active installer")
	}
	// Cancel pending installs of the version we advanced away from.
	if err := ds.ProcessInstallerUpdateSideEffects(ctx, c.InstallerID, true, false); err != nil {
		return ctxerr.Wrap(ctx, err, "processing installer update side effects")
	}

	logger.InfoContext(ctx, "advanced fleet-maintained app to newer cached version",
		"title_id", c.TitleID, "team_id", teamIDForLog(c.TeamID), "slug", c.Slug,
		"from", c.Version, "to", target.Version, "pin", pin)
	return nil
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
