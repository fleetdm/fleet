package vpp

import (
	"context"
	"sort"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
)

// RefreshVersions updates the LatestVersion fields for the VPP apps stored in
// Fleet. It groups apps by their anchored storefront (country_code on
// vpp_apps), then for each group picks a token-of-that-country that owns at
// least one of the apps and bundles all such apps into one Apple call. Apps
// whose anchored country has no eligible token in Fleet are skipped silently
// — their stored versions stay until a token is uploaded for that country.
func RefreshVersions(ctx context.Context, ds fleet.Datastore, vppAppsConfig apple_apps.Config) error {
	apps, err := ds.GetAllVPPApps(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting all VPP apps")
	}
	if len(apps) == 0 {
		return nil
	}

	tokens, err := ds.ListVPPTokens(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting all VPP tokens")
	}

	// Index apps by adamID so we can fan a single Apple response back out
	// across the per-platform rows that share an adamID.
	appsByAdamID := make(map[string][]*fleet.VPPApp, len(apps))
	for _, app := range apps {
		appsByAdamID[app.AdamID] = append(appsByAdamID[app.AdamID], app)
	}

	// Group adamIDs by anchored country. Apps with an empty country (e.g.
	// pre-migration rows that haven't been re-anchored yet) cannot be
	// refreshed without guessing the storefront, so they are skipped.
	adamIDsByCountry := make(map[string]map[string]struct{})
	for adamID, group := range appsByAdamID {
		country := group[0].CountryCode
		if country == "" {
			continue
		}
		if adamIDsByCountry[country] == nil {
			adamIDsByCountry[country] = make(map[string]struct{})
		}
		adamIDsByCountry[country][adamID] = struct{}{}
	}

	// For each country, pick exactly one token (lowest token id) that owns
	// at least one of the country's apps and bundle the adamIDs that share
	// that token. AdamIDs whose owning tokens are not represented in this
	// country are skipped silently.
	type bundle struct {
		token   *fleet.VPPTokenDB
		adamIDs []string
	}
	var bundles []bundle

	for country, adamSet := range adamIDsByCountry {
		// Find candidate tokens: same country, ordered by id.
		var candidates []*fleet.VPPTokenDB
		for _, t := range tokens {
			if t.CountryCode == country {
				candidates = append(candidates, t)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })

		remaining := make(map[string]struct{}, len(adamSet))
		for adamID := range adamSet {
			remaining[adamID] = struct{}{}
		}

		// For each candidate token, in deterministic order, ask Apple for
		// the apps it owns (assets list). We bundle the apps it can refresh.
		for _, tok := range candidates {
			if len(remaining) == 0 {
				break
			}
			assets, err := GetAssets(ctx, tok.Token, nil)
			if err != nil {
				// Skip this token; try the next candidate. Refresh is
				// best-effort.
				continue
			}
			ownedSet := make(map[string]struct{}, len(assets))
			for _, a := range assets {
				ownedSet[a.AdamID] = struct{}{}
			}

			var bundleAdams []string
			for adamID := range remaining {
				if _, ok := ownedSet[adamID]; ok {
					bundleAdams = append(bundleAdams, adamID)
					delete(remaining, adamID)
				}
			}

			if len(bundleAdams) > 0 {
				sort.Strings(bundleAdams)
				bundles = append(bundles, bundle{token: tok, adamIDs: bundleAdams})
			}
		}
	}

	if len(bundles) == 0 {
		return nil
	}

	// Run the metadata fetches and collect updates.
	var appsToUpdate []*fleet.VPPApp
	for _, b := range bundles {
		meta, err := apple_apps.GetMetadata(b.adamIDs, b.token.CountryCode, b.token.Token, vppAppsConfig)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting VPP app metadata from Apple API")
		}

		for adamID, m := range meta {
			retrievedByPlatform := apple_apps.ToVPPApps(m)
			for _, app := range appsByAdamID[adamID] {
				current, ok := retrievedByPlatform[app.Platform]
				if !ok {
					continue
				}
				if current.LatestVersion != app.LatestVersion ||
					current.Name != app.Name ||
					current.IconURL != app.IconURL {
					app.LatestVersion = current.LatestVersion
					app.Name = current.Name
					app.IconURL = current.IconURL
					appsToUpdate = append(appsToUpdate, app)
				}
			}
		}
	}

	if len(appsToUpdate) == 0 {
		return nil
	}

	if err := ds.InsertVPPApps(ctx, appsToUpdate); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting VPP apps with new versions")
	}

	return nil
}
