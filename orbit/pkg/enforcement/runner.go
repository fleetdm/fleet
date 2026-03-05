package enforcement

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// EnforcementClient is the interface for reporting results back to Fleet.
type EnforcementClient interface {
	// SetOrUpdateDeviceMapping sets the enforcement status for a host.
}

// WindowsEnforcementRunner is a ConfigReceiver that enforces Windows security
// policies received via OrbitConfig. It follows the BitLocker receiver pattern.
type WindowsEnforcementRunner struct {
	handlers map[string]Handler
	cache    *ComplianceCache
	lastHash string
	mu       sync.Mutex
}

// NewRunner creates a new WindowsEnforcementRunner with the given handlers.
func NewRunner(handlers map[string]Handler, cache *ComplianceCache) *WindowsEnforcementRunner {
	return &WindowsEnforcementRunner{
		handlers: handlers,
		cache:    cache,
	}
}

// Run implements fleet.OrbitConfigReceiver. It checks if the enforcement hash
// has changed and, if so, applies the enforcement policies.
func (r *WindowsEnforcementRunner) Run(cfg *fleet.OrbitConfig) error {
	hash := cfg.Notifications.PendingWindowsEnforcementHash
	if hash == "" || hash == r.lastHash {
		return nil
	}

	if !r.mu.TryLock() {
		return nil // another enforcement run is in progress
	}
	defer r.mu.Unlock()

	if cfg.WindowsEnforcement == nil || len(cfg.WindowsEnforcement.Policies) == 0 {
		r.lastHash = hash
		return nil
	}

	log.Info().Int("count", len(cfg.WindowsEnforcement.Policies)).Str("hash", hash).Msg("applying enforcement policies")

	var allRecords []ComplianceRecord
	now := time.Now()

	for _, policy := range cfg.WindowsEnforcement.Policies {
		for _, handler := range r.handlers {
			// Run diff to get compliance state
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			results, err := handler.Diff(ctx, policy.RawPolicy)
			cancel()

			if err != nil {
				log.Warn().Err(err).Str("handler", handler.Name()).Str("policy", policy.Name).Msg("enforcement diff failed")
				continue
			}

			// Check for non-compliant settings and apply
			var needsApply bool
			for _, result := range results {
				if !result.Compliant {
					needsApply = true
					break
				}
			}

			if needsApply {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				applyResults, err := handler.Apply(ctx, policy.RawPolicy)
				cancel()

				if err != nil {
					log.Warn().Err(err).Str("handler", handler.Name()).Str("policy", policy.Name).Msg("enforcement apply failed")
				} else {
					for _, ar := range applyResults {
						if !ar.Success {
							log.Warn().Str("handler", handler.Name()).Str("setting", ar.SettingName).Str("error", ar.Error).Msg("enforcement setting failed")
						}
					}
				}

				// Re-diff after apply to get final compliance state
				ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
				results, err = handler.Diff(ctx2, policy.RawPolicy)
				cancel2()

				if err != nil {
					log.Warn().Err(err).Str("handler", handler.Name()).Str("policy", policy.Name).Msg("post-apply diff failed")
					continue
				}
			}

			// Collect compliance records
			for _, result := range results {
				allRecords = append(allRecords, ComplianceRecord{
					SettingName:  result.SettingName,
					Category:     result.Category,
					PolicyName:   policy.Name,
					CISRef:       result.CISRef,
					DesiredValue: result.DesiredValue,
					CurrentValue: result.CurrentValue,
					Compliant:    result.Compliant,
					LastChecked:  now,
				})
			}
		}
	}

	// Update the compliance cache for the osquery table
	r.cache.Update(allRecords)
	r.lastHash = hash

	log.Info().Int("records", len(allRecords)).Str("hash", hash).Msg("enforcement complete")

	return nil
}
