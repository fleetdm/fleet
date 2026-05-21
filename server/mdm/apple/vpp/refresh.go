package vpp

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
)

// RefreshVersions updates the LatestVersion fields for the VPP apps stored in
// Fleet. For each (adam_id, platform) row it picks a token to refresh from,
// preferring a token of that row's anchored country (the storefront the row
// was originally added from). If no token of the anchored country owns the
// app, it falls back to any other token that owns it and re-anchors the row
// to that token's country, so future refreshes continue to work. Picking is
// per-row because two platforms of the same adam_id can have diverged
// anchors. Apps that no remaining token owns are skipped silently and keep
// their last-known metadata.
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
	slices.SortFunc(tokens, func(a, b *fleet.VPPTokenDB) int { return cmp.Compare(a.ID, b.ID) })

	// Anchoring is per (adam_id, platform), so picks and bundles key on the
	// pair. The asset list from Apple is per-adamID, but two platforms of
	// the same adamID may resolve to different tokens.
	type appKey struct {
		adamID   string
		platform fleet.InstallableDevicePlatform
	}
	appsByKey := make(map[appKey]*fleet.VPPApp, len(apps))
	for _, app := range apps {
		appsByKey[appKey{adamID: app.AdamID, platform: app.Platform}] = app
	}

	// Cache GetAssets results so we don't ask the same token twice. On
	// error we cache the error rather than an empty set, so the picks loop
	// can distinguish "this token doesn't own the app" from "we don't
	// know" and avoid an incorrect cross-country re-anchor.
	type ownedResult struct {
		owned map[string]struct{}
		err   error
	}
	ownedByToken := make(map[uint]ownedResult, len(tokens))
	getOwned := func(tok *fleet.VPPTokenDB) ownedResult {
		if r, ok := ownedByToken[tok.ID]; ok {
			return r
		}
		assets, err := GetAssets(ctx, tok.Token, nil)
		if err != nil {
			r := ownedResult{err: err}
			ownedByToken[tok.ID] = r
			return r
		}
		owned := make(map[string]struct{}, len(assets))
		for _, a := range assets {
			owned[a.AdamID] = struct{}{}
		}
		r := ownedResult{owned: owned}
		ownedByToken[tok.ID] = r
		return r
	}

	// For each app row with a non-empty anchored country, pick exactly one
	// token to refresh from. Prefer tokens of the row's anchored country;
	// fall back to any other token that owns the app.
	type pick struct {
		token *fleet.VPPTokenDB
	}
	picks := make(map[appKey]pick)

	for k, app := range appsByKey {
		anchored := app.CountryCode
		if anchored == "" {
			continue
		}

		// First pass: tokens of the anchored country.
		anchoredErrored := false
		for _, t := range tokens {
			if t.CountryCode != anchored {
				continue
			}
			r := getOwned(t)
			if r.err != nil {
				anchoredErrored = true
				continue
			}
			if _, ok := r.owned[k.adamID]; ok {
				picks[k] = pick{token: t}
				break
			}
		}
		if _, ok := picks[k]; ok {
			continue
		}
		if anchoredErrored {
			// Anchored-country tokens errored, so we can't be sure none of
			// them own this row. Skip fallback to avoid an incorrect
			// re-anchor; retry on the next refresh.
			continue
		}

		// Fallback: any other token that owns the app. Will re-anchor the
		// row to that token's country.
		for _, t := range tokens {
			if t.CountryCode == anchored || t.CountryCode == "" {
				continue
			}
			r := getOwned(t)
			if r.err != nil {
				continue
			}
			if _, ok := r.owned[k.adamID]; ok {
				picks[k] = pick{token: t}
				break
			}
		}
	}

	if len(picks) == 0 {
		return nil
	}

	// Bundle picks by token id, tracking the (adamID, platform) keys per
	// bundle. The Apple call is by token+region with deduped adamIDs; the
	// metadata loop dispatches per-platform back to the picked rows.
	type bundle struct {
		token   *fleet.VPPTokenDB
		keys    []appKey
		adamIDs []string
	}
	bundlesByTokenID := make(map[uint]*bundle)
	for k, p := range picks {
		b, ok := bundlesByTokenID[p.token.ID]
		if !ok {
			b = &bundle{token: p.token}
			bundlesByTokenID[p.token.ID] = b
		}
		b.keys = append(b.keys, k)
	}

	bundles := make([]*bundle, 0, len(bundlesByTokenID))
	for _, b := range bundlesByTokenID {
		seen := make(map[string]struct{}, len(b.keys))
		for _, k := range b.keys {
			if _, dup := seen[k.adamID]; dup {
				continue
			}
			seen[k.adamID] = struct{}{}
			b.adamIDs = append(b.adamIDs, k.adamID)
		}
		slices.Sort(b.adamIDs)
		bundles = append(bundles, b)
	}
	slices.SortFunc(bundles, func(a, b *bundle) int { return cmp.Compare(a.token.ID, b.token.ID) })

	var appsToUpdate []*fleet.VPPApp
	var bundleErrs []error
	for _, b := range bundles {
		meta, err := apple_apps.GetMetadata(b.adamIDs, b.token.CountryCode, b.token.Token, vppAppsConfig)
		if err != nil {
			// A flaky storefront must not block metadata updates for
			// healthy storefronts.
			bundleErrs = append(bundleErrs, ctxerr.Wrap(ctx, err, "getting VPP app metadata from Apple API"))
			continue
		}

		for _, k := range b.keys {
			m, ok := meta[k.adamID]
			if !ok {
				continue
			}
			current, ok := apple_apps.ToVPPApps(m)[k.platform]
			if !ok {
				continue
			}

			// Apple occasionally returns blanks for transiently-degraded
			// apps, so don't overwrite stored metadata with empty values.
			if current.Name == "" || current.LatestVersion == "" || current.IconURL == "" {
				continue
			}

			app := appsByKey[k]
			if app == nil {
				continue
			}

			// Re-anchor only the picked row when the picked token's country
			// differs from the row's stored country. Sibling platforms of
			// the same adam_id with their own anchors are untouched.
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

	if len(appsToUpdate) > 0 {
		if err := ds.InsertVPPApps(ctx, appsToUpdate); err != nil {
			bundleErrs = append(bundleErrs, ctxerr.Wrap(ctx, err, "inserting VPP apps with new versions"))
		}
	}

	if len(bundleErrs) > 0 {
		return errors.Join(bundleErrs...)
	}
	return nil
}
