package vpp

import (
	"context"
	"sort"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
)

// RefreshVersions updates the LatestVersion fields for the VPP apps stored in
// Fleet. For each app it picks a token to refresh from, preferring a token of
// the app's anchored country (the storefront the app was originally added
// from). If no token of the anchored country owns the app — for example, the
// original token was deleted from Fleet — it falls back to any other token
// that owns the app and re-anchors the app to that token's country, so future
// refreshes continue to work. Apps that no remaining token owns are skipped
// silently; they keep their last-known metadata until a token is uploaded
// that does own them.
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

	// Iterate tokens in deterministic order so the picks below are stable.
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].ID < tokens[j].ID })

	// Index apps by adamID so we can fan a single Apple response back out
	// across the per-platform rows that share an adamID.
	appsByAdamID := make(map[string][]*fleet.VPPApp, len(apps))
	for _, app := range apps {
		appsByAdamID[app.AdamID] = append(appsByAdamID[app.AdamID], app)
	}

	// Cache GetAssets results so we don't ask the same token twice — once for
	// its own anchored country group, once during the cross-country fallback.
	ownedByToken := make(map[uint]map[string]struct{}, len(tokens))
	getOwned := func(tok *fleet.VPPTokenDB) map[string]struct{} {
		if owned, ok := ownedByToken[tok.ID]; ok {
			return owned
		}
		assets, err := GetAssets(ctx, tok.Token, nil)
		if err != nil {
			// Best effort: cache an empty set so we don't retry this token
			// repeatedly within one refresh run.
			ownedByToken[tok.ID] = map[string]struct{}{}
			return ownedByToken[tok.ID]
		}
		owned := make(map[string]struct{}, len(assets))
		for _, a := range assets {
			owned[a.AdamID] = struct{}{}
		}
		ownedByToken[tok.ID] = owned
		return owned
	}

	// For each adamID with a non-empty anchored country, pick exactly one
	// token to refresh from. Prefer tokens of the anchored country; fall
	// back to any other token that owns the app.
	type pick struct {
		token *fleet.VPPTokenDB
	}
	picks := make(map[string]pick)

	for adamID, group := range appsByAdamID {
		anchored := group[0].CountryCode
		if anchored == "" {
			continue
		}

		// First pass: tokens of the anchored country.
		for _, t := range tokens {
			if t.CountryCode != anchored {
				continue
			}
			if _, ok := getOwned(t)[adamID]; ok {
				picks[adamID] = pick{token: t}
				break
			}
		}
		if _, ok := picks[adamID]; ok {
			continue
		}

		// Fallback: any other token that owns the app. Will re-anchor the
		// app to that token's country.
		for _, t := range tokens {
			if t.CountryCode == anchored || t.CountryCode == "" {
				continue
			}
			if _, ok := getOwned(t)[adamID]; ok {
				picks[adamID] = pick{token: t}
				break
			}
		}
	}

	if len(picks) == 0 {
		return nil
	}

	// Bundle picks by token to minimize Apple metadata calls.
	type bundle struct {
		token   *fleet.VPPTokenDB
		adamIDs []string
	}
	bundlesByTokenID := make(map[uint]*bundle)
	for adamID, p := range picks {
		b, ok := bundlesByTokenID[p.token.ID]
		if !ok {
			b = &bundle{token: p.token}
			bundlesByTokenID[p.token.ID] = b
		}
		b.adamIDs = append(b.adamIDs, adamID)
	}

	orderedTokenIDs := make([]uint, 0, len(bundlesByTokenID))
	for id := range bundlesByTokenID {
		orderedTokenIDs = append(orderedTokenIDs, id)
	}
	sort.Slice(orderedTokenIDs, func(i, j int) bool { return orderedTokenIDs[i] < orderedTokenIDs[j] })

	var appsToUpdate []*fleet.VPPApp
	for _, id := range orderedTokenIDs {
		b := bundlesByTokenID[id]
		sort.Strings(b.adamIDs)

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

				// If we picked a token whose country differs from the app's
				// anchored country, re-anchor so future refreshes pick this
				// token's country directly.
				if app.CountryCode != b.token.CountryCode {
					if err := ds.UpdateVPPAppCountryCode(ctx, app.AdamID, app.Platform, b.token.CountryCode); err != nil {
						return ctxerr.Wrap(ctx, err, "re-anchoring VPP app country")
					}
					app.CountryCode = b.token.CountryCode
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
