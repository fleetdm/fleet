package gdmf

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// getAssetMetadataFn is overridden in tests.
var getAssetMetadataFn = GetAssetMetadataWithContext

// SyncMacOSCurrencyPolicies fetches Apple's GDMF feed, refreshes
// apple_software_update_assets for macOS, and rewrites Fleet-managed macOS
// OS-currency policy queries from the resulting version floors.
func SyncMacOSCurrencyPolicies(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, now time.Time) error {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "gdmf-macos-currency")

	meta, err := getAssetMetadataFn(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetch GDMF asset metadata")
	}

	candidateAssets := MacOSAssetsForCurrencyPolicies(meta)
	if len(candidateAssets) == 0 {
		// Preserve last-known-good cache when Apple returns an empty set.
		logger.InfoContext(ctx, "no macOS assets in GDMF response; preserving cache and skipping policy refresh")
		return nil
	}

	assets := usableMacOSAssets(candidateAssets)
	if len(assets) == 0 {
		return ctxerr.New(ctx, "GDMF response contains no usable macOS assets; preserving cache and skipping policy refresh")
	}

	type policyUpdate struct {
		policy MacOSCurrencyPolicy
		query  string
	}
	policies := MacOSCurrencyPolicies()
	policyUpdates := make([]policyUpdate, 0, len(policies))
	for _, p := range policies {
		floors := RequiredMacOSVersions(assets, p.GraceDays, now)
		query := PolicyQuery(floors)
		if query == "" {
			return ctxerr.Errorf(ctx, "could not generate managed macOS currency policy query for %q; preserving cache and skipping policy refresh", p.Key)
		}
		policyUpdates = append(policyUpdates, policyUpdate{policy: p, query: query})
	}

	if err := replaceMacOSAssets(ctx, ds, meta, assets); err != nil {
		return err
	}
	logger.InfoContext(ctx, "refreshed apple_software_update_assets from GDMF",
		"macos_assets", len(assets),
	)

	for _, update := range policyUpdates {
		ids, err := ds.UpdateFleetManagedPolicyQueries(ctx, update.policy.Key, update.query)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Fleet-managed macOS currency policy queries")
		}
		if len(ids) > 0 {
			logger.InfoContext(ctx, "updated Fleet-managed macOS currency policy queries",
				"fleet_managed_key", update.policy.Key,
				"grace_days", update.policy.GraceDays,
				"query", update.query,
				"updated_count", len(ids),
			)
		}
	}
	return nil
}

func usableMacOSAssets(src []Asset) []Asset {
	assets := make([]Asset, 0, len(src))
	for _, asset := range src {
		if _, ok := majorVersion(asset.ProductVersion); !ok || !validOSVersion(asset.ProductVersion) {
			continue
		}
		assets = append(assets, asset)
	}
	return assets
}

func replaceMacOSAssets(ctx context.Context, ds fleet.Datastore, meta *AssetMetadata, src []Asset) error {
	if meta == nil {
		return ctxerr.New(ctx, "GDMF asset metadata is nil")
	}
	if len(src) == 0 {
		return ctxerr.New(ctx, "refusing to replace apple software update assets with empty set")
	}

	assets := make([]fleet.AppleSoftwareUpdateAsset, 0, len(src))
	seen := map[string]struct{}{}
	for _, a := range src {
		key := a.ProductVersion + "\x00" + a.Build
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		row := fleet.AppleSoftwareUpdateAsset{
			Class:          fleet.AppleSoftwareUpdateAssetClassMacOS,
			ProductVersion: a.ProductVersion,
			Build:          a.Build,
		}
		if t, ok := parsePostingDate(a.PostingDate); ok {
			row.PostingDate = &t
		}
		if t, ok := parsePostingDate(a.ExpirationDate); ok {
			row.ExpirationDate = &t
		}
		devices, err := json.Marshal(a.SupportedDevices)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal supported devices")
		}
		if len(a.SupportedDevices) == 0 {
			devices = []byte("[]")
		}
		row.SupportedDevices = devices
		assets = append(assets, row)
	}
	if err := ds.ReplaceAppleSoftwareUpdateAssets(ctx, fleet.AppleSoftwareUpdateAssetClassMacOS, assets); err != nil {
		return ctxerr.Wrap(ctx, err, "replace apple software update assets")
	}
	return nil
}
