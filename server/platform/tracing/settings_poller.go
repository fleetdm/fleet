package tracing

import (
	"context"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// settingsPollInterval is how often each Fleet replica re-reads
// trace_sampler_settings. 60s matches industry defaults for feature-flag-style
// runtime config (LaunchDarkly polling SDK, GrowthBook, DataDog agent refresh,
// Consul template are all in the 30-60s range). The row changes maybe once a
// quarter — during incident debugging — so sub-minute polling would be
// dominated by waste.
const settingsPollInterval = 60 * time.Second

// settingsReader is the minimal datastore surface the poller needs. The full
// fleet.Datastore is large; depending only on this interface keeps the
// poller's tests cheap.
type settingsReader interface {
	GetTraceSamplerSettings(ctx context.Context) (*fleet.TraceSamplerSettings, error)
}

// StartSettingsPoller runs the polling loop until ctx is cancelled. On each
// tick it reads the trace_sampler_settings row, compares the values to the
// last-applied state, and calls sampler.Apply only when something changed.
// This matches the read-compare-apply pattern at
// server/service/schedule/schedule.go:401-446.
//
// On startup it does one immediate read so the sampler picks up the row's
// current values without waiting a full interval. If the first read fails,
// the sampler keeps its compile-time defaults and a warning is logged; the
// next tick will try again.
func StartSettingsPoller(ctx context.Context, sampler *RouteTierSampler, ds settingsReader, logger *slog.Logger) {
	pollAndApply := func(last *fleet.TraceSamplerSettings) *fleet.TraceSamplerSettings {
		got, err := ds.GetTraceSamplerSettings(ctx)
		if err != nil {
			logger.WarnContext(ctx, "trace sampler settings poll failed", "err", err)
			return last
		}
		if last != nil &&
			got.HighVolumeRatio == last.HighVolumeRatio &&
			got.StandardRatio == last.StandardRatio &&
			got.ForceFull == last.ForceFull {
			return last
		}
		sampler.Apply(got.HighVolumeRatio, got.StandardRatio, got.ForceFull)
		logger.InfoContext(ctx, "trace sampler settings applied",
			"high_volume_ratio", got.HighVolumeRatio,
			"standard_ratio", got.StandardRatio,
			"force_full", got.ForceFull,
		)
		return got
	}

	var last *fleet.TraceSamplerSettings
	last = pollAndApply(last)

	ticker := time.NewTicker(settingsPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			last = pollAndApply(last)
		}
	}
}
