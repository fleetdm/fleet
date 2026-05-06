package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	chart_api "github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// Job names for chart scrub workers. Persisted in the jobs table as the
// `name` column, so changing them is a wire break — keep stable.
const (
	ChartScrubDatasetGlobalJobName = "chart_scrub_dataset_global"
	ChartScrubDatasetFleetJobName  = "chart_scrub_dataset_fleet"
)

// ChartScrubGlobalArgs is the payload for chart_scrub_dataset_global jobs.
type ChartScrubGlobalArgs struct {
	Dataset string `json:"dataset"`
}

// ChartScrubFleetArgs is the payload for chart_scrub_dataset_fleet jobs.
// FleetIDs is always populated with at least one entry; today, enqueue
// creates one job per team flip — coalescing across teams is a future optimization.
type ChartScrubFleetArgs struct {
	Dataset  string `json:"dataset"`
	FleetIDs []uint `json:"fleet_ids"`
}

// ChartScrubGlobal runs the global scrub job: deletes every host_scd_data row
// for the configured dataset.
type ChartScrubGlobal struct {
	ChartService chart_api.Service
	Log          *slog.Logger
}

func (c *ChartScrubGlobal) Name() string { return ChartScrubDatasetGlobalJobName }

func (c *ChartScrubGlobal) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args ChartScrubGlobalArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal chart scrub global args")
	}
	if args.Dataset == "" {
		return ctxerr.New(ctx, "chart scrub global: empty dataset")
	}
	if c.Log != nil {
		c.Log.InfoContext(ctx, "chart scrub global: starting", "dataset", args.Dataset)
	}
	if err := c.ChartService.ScrubDatasetGlobal(ctx, args.Dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "scrub dataset global")
	}
	if c.Log != nil {
		c.Log.InfoContext(ctx, "chart scrub global: complete", "dataset", args.Dataset)
	}
	return nil
}

// ChartScrubFleet runs the per-fleet scrub job: clears bits for every host
// currently in any of the listed fleets from existing host_scd_data rows for
// the configured dataset.
type ChartScrubFleet struct {
	ChartService chart_api.Service
	Log          *slog.Logger
}

func (c *ChartScrubFleet) Name() string { return ChartScrubDatasetFleetJobName }

func (c *ChartScrubFleet) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args ChartScrubFleetArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal chart scrub fleet args")
	}
	if args.Dataset == "" {
		return ctxerr.New(ctx, "chart scrub fleet: empty dataset")
	}
	if len(args.FleetIDs) == 0 {
		// Defensive — enqueue side should never produce this. Treat as a
		// no-op rather than scanning the entire table.
		if c.Log != nil {
			c.Log.WarnContext(ctx, "chart scrub fleet: empty fleet list", "dataset", args.Dataset)
		}
		return nil
	}
	if c.Log != nil {
		c.Log.InfoContext(ctx, "chart scrub fleet: starting", "dataset", args.Dataset, "fleet_ids", args.FleetIDs)
	}
	if err := c.ChartService.ScrubDatasetFleet(ctx, args.Dataset, args.FleetIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "scrub dataset fleet")
	}
	if c.Log != nil {
		c.Log.InfoContext(ctx, "chart scrub fleet: complete", "dataset", args.Dataset)
	}
	return nil
}

