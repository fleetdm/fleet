package tracing

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// settingsPollInterval is how often each Fleet replica re-reads trace_sampler_settings. 60s matches industry defaults for
// feature flag style runtime config.
const settingsPollInterval = 60 * time.Second

// pollTracer instruments the poll loop. When OTEL is not configured the global provider returns a no-op tracer, so this is
// free when tracing is disabled.
var pollTracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/platform/tracing")

// settingsReader is the minimal datastore surface the poller needs. The full fleet.Datastore is large. Depending only on this
// interface keeps the poller's tests cheap and the package free of cross context coupling.
type settingsReader interface {
	GetTraceSamplerSettings(ctx context.Context) (*Settings, error)
}

// StartSettingsPoller runs the polling loop until ctx is cancelled. On each tick it reads the trace_sampler_settings row,
// compares the values to the last applied state, and calls sampler. Apply only when something changed.
//
// On startup it does one immediate read so the sampler picks up the row's current values without waiting a full interval. If
// the first read fails, the sampler keeps its compile time defaults and a warning is logged. The next tick will try again.
func StartSettingsPoller(ctx context.Context, sampler *RouteTierSampler, ds settingsReader, logger *slog.Logger) {
	pollAndApply := func(last *Settings) *Settings {
		spanCtx, span := pollTracer.Start(ctx, "tracing.poll_settings",
			trace.WithNewRoot(),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()

		got, err := ds.GetTraceSamplerSettings(spanCtx)
		if err != nil {
			logger.ErrorContext(spanCtx, "trace sampler settings poll failed", "err", err)
			return last
		}
		if last != nil &&
			got.HighVolumeRatio == last.HighVolumeRatio &&
			got.StandardRatio == last.StandardRatio &&
			got.ForceFull == last.ForceFull {
			return last
		}
		sampler.Apply(got.HighVolumeRatio, got.StandardRatio, got.ForceFull)
		logger.InfoContext(spanCtx, "trace sampler settings applied",
			"high_volume_ratio", got.HighVolumeRatio,
			"standard_ratio", got.StandardRatio,
			"force_full", got.ForceFull,
		)
		return got
	}

	var last *Settings
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
