package vpp

import (
	"context"
	"log/slog"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// BackfillLegacyCountries is a one-shot helper that fills `country_code` for
// VPP tokens and apps that predate the multi-region storefront feature.
//
// Sequence:
//  1. List all VPP tokens. If none have a NULL country, return immediately.
//  2. For each NULL-country token, call Apple's /client/config in parallel,
//     persist the returned country. Failures are logged and the column stays
//     NULL — the next boot retries.
//  3. After tokens are populated, run a single SQL UPDATE that copies
//     each token's country onto associated `vpp_apps` rows that still have
//     NULL country (joined via `vpp_apps_teams.vpp_token_id`).
//
// Designed to run in a goroutine at server startup so it doesn't block boot.
// After every existing customer has booted at least one version with this
// helper, the function becomes a permanent no-op (steps 1 and 3 both touch
// zero rows) and is safe to remove. See FIXME below.
//
// FIXME(43846): remove this helper and its caller in cmd/fleet/serve.go once
// Fleet versions <= the release that introduced country_code are no longer
// supported. The lazy fallback in ee/server/service/vpp.go ensureVPPTokenCountry
// can be deleted at the same time.
func BackfillLegacyCountries(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) {
	tokens, err := ds.ListVPPTokens(ctx)
	if err != nil {
		logger.WarnContext(ctx, "vpp legacy backfill: list tokens failed", "err", err)
		return
	}

	var needsBackfill []*fleet.VPPTokenDB
	for _, t := range tokens {
		if t.CountryCode == "" {
			needsBackfill = append(needsBackfill, t)
		}
	}

	if len(needsBackfill) > 0 {
		logger.InfoContext(ctx, "vpp legacy backfill: backfilling token countries",
			"count", len(needsBackfill))

		var wg sync.WaitGroup
		var filled int64
		var filledMu sync.Mutex

		// Cap concurrency so a tenant with many tokens doesn't burst Apple's
		// /client/config endpoint at startup.
		const maxConcurrent = 8
		sem := make(chan struct{}, maxConcurrent)

		for _, t := range needsBackfill {
			wg.Add(1)
			sem <- struct{}{}
			go func(token *fleet.VPPTokenDB) {
				defer wg.Done()
				defer func() { <-sem }()

				cfg, err := GetConfig(ctx, token.Token)
				if err != nil {
					logger.WarnContext(ctx, "vpp legacy backfill: GetConfig failed",
						"vpp_token_id", token.ID,
						"err", err)
					return
				}
				if cfg.CountryCode == "" {
					logger.WarnContext(ctx, "vpp legacy backfill: empty country in /client/config",
						"vpp_token_id", token.ID)
					return
				}
				if err := ds.UpdateVPPTokenCountryCode(ctx, token.ID, cfg.CountryCode); err != nil {
					logger.WarnContext(ctx, "vpp legacy backfill: persist failed",
						"vpp_token_id", token.ID,
						"err", err)
					return
				}

				filledMu.Lock()
				filled++
				filledMu.Unlock()
			}(t)
		}
		wg.Wait()

		logger.InfoContext(ctx, "vpp legacy backfill: token country backfill complete",
			"filled", filled,
			"total", len(needsBackfill))
	}

	// Step 3 — run regardless. If no tokens were filled this pass but a
	// previous boot already filled them, the SQL still finds vpp_apps rows
	// to update. After everything is populated it's a no-op.
	rows, err := ds.BackfillVPPAppCountriesFromTokens(ctx)
	if err != nil {
		logger.WarnContext(ctx, "vpp legacy backfill: app country backfill failed",
			"err", err)
		return
	}
	if rows > 0 {
		logger.InfoContext(ctx, "vpp legacy backfill: app countries filled",
			"rows", rows)
	}
}
