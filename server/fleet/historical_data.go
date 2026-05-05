package fleet

import (
	"context"
	"encoding/json"
	"fmt"
)

// HistoricalDataScrubEnqueuer is the narrow interface needed by
// EnqueueHistoricalDataScrubs. The fleet.Datastore satisfies it via NewJob
// and HasQueuedJobWithArgs. Defined locally to avoid pulling worker-package
// types into the fleet package.
//
// HasQueuedJobWithArgs gates the NewJob call so rapid disable/enable toggles
// of the same scope don't stack identical scrub jobs in the queue. See
// design decision 5 of the chart-disabling-collection-scrub change.
type HistoricalDataScrubEnqueuer interface {
	NewJob(ctx context.Context, j *Job) (*Job, error)
	HasQueuedJobWithArgs(ctx context.Context, name string, args json.RawMessage) (bool, error)
}

// HistoricalDataActivityEmitter is the narrow interface needed by
// OnHistoricalDataChanged. Both the free service and the EE service
// satisfy it via their NewActivity method.
type HistoricalDataActivityEmitter interface {
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
}

// OnHistoricalDataChanged is the hook called when historical_data config changes.
// It emits one activity per historical_data sub-key whose value differs between
// oldHD and newHD. fleetID and fleetName are nil for global toggles and populated
// for per-fleet toggles. Dataset names in the activity payload are the public
// config sub-keys ("uptime", "vulnerabilities"), not internal dataset names.
func OnHistoricalDataChanged(
	ctx context.Context,
	emitter HistoricalDataActivityEmitter,
	user *User,
	oldHD, newHD HistoricalDataSettings,
	fleetID *uint, fleetName *string,
) error {
	changes := []struct {
		dataset string
		oldVal  bool
		newVal  bool
	}{
		{"uptime", oldHD.Uptime, newHD.Uptime},
		{"vulnerabilities", oldHD.Vulnerabilities, newHD.Vulnerabilities},
	}
	for _, c := range changes {
		if c.oldVal == c.newVal {
			continue
		}
		var act ActivityDetails
		if c.newVal {
			act = ActivityTypeEnabledHistoricalDataset{
				Dataset:   c.dataset,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		} else {
			act = ActivityTypeDisabledHistoricalDataset{
				Dataset:   c.dataset,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		}
		if err := emitter.NewActivity(ctx, user, act); err != nil {
			return err
		}
	}
	return nil
}

// Worker job names mirrored here to avoid an import cycle between fleet
// and server/worker. If these strings drift from the worker constants,
// jobs will be enqueued under one name and never picked up. The worker
// package's chart_scrub.go is the source of truth.
const (
	chartScrubDatasetGlobalJobName = "chart_scrub_dataset_global"
	chartScrubDatasetFleetJobName  = "chart_scrub_dataset_fleet"
)

// chartScrubGlobalArgs and chartScrubFleetArgs mirror the payload structs in
// server/worker/chart_scrub.go. Declared locally for the same import-cycle
// reason as the job-name constants above. The JSON shape is the contract;
// keep field names and tags in sync.
type chartScrubGlobalArgs struct {
	Dataset string `json:"dataset"`
}

type chartScrubFleetArgs struct {
	Dataset  string `json:"dataset"`
	FleetIDs []uint `json:"fleet_ids"`
}

// EnqueueHistoricalDataScrubs inserts scrub jobs into the jobs table for any
// historical_data sub-key that flipped from true to false between oldHD and
// newHD. Mirror of OnHistoricalDataChanged, but for the side effect of
// removing already-collected data when an admin disables a dataset.
//
// fleetID semantics:
//   - nil: a global flip. Each disabled dataset enqueues one
//     chart_scrub_dataset_global job that will DELETE every row.
//   - non-nil: a per-fleet flip. Each disabled dataset enqueues one
//     chart_scrub_dataset_fleet job whose fleet_ids slice contains just
//     this fleet ID. (Per-call coalescing across multiple teams within a
//     single batch is a future optimization — see the
//     chart-historical-data-collection spec.)
//
// CALLER ORDERING REQUIREMENT: this function MUST be called AFTER the
// corresponding SaveAppConfig / SaveTeam commit has succeeded. Enqueuing
// before commit risks a worker picking up the job before the new config is
// visible, which would race the collection cron and re-introduce the bits
// the scrub is meant to clear.
func EnqueueHistoricalDataScrubs(
	ctx context.Context,
	enq HistoricalDataScrubEnqueuer,
	oldHD, newHD HistoricalDataSettings,
	fleetID *uint,
) error {
	changes := []struct {
		// scrubDataset is the internal dataset name (matches host_scd_data.dataset
		// and the chart-package Dataset.Name() return values).
		scrubDataset string
		oldVal       bool
		newVal       bool
	}{
		{"uptime", oldHD.Uptime, newHD.Uptime},
		{"cve", oldHD.Vulnerabilities, newHD.Vulnerabilities},
	}
	for _, c := range changes {
		if c.oldVal == c.newVal || c.newVal /* false → true: no scrub */ {
			continue
		}

		var jobName string
		var argsJSON []byte
		var err error
		if fleetID == nil {
			jobName = chartScrubDatasetGlobalJobName
			argsJSON, err = json.Marshal(chartScrubGlobalArgs{Dataset: c.scrubDataset})
		} else {
			jobName = chartScrubDatasetFleetJobName
			argsJSON, err = json.Marshal(chartScrubFleetArgs{
				Dataset:  c.scrubDataset,
				FleetIDs: []uint{*fleetID},
			})
		}
		if err != nil {
			return fmt.Errorf("marshal scrub job args for %s: %w", c.scrubDataset, err)
		}
		raw := json.RawMessage(argsJSON)

		// Dedup: if an identical job is already queued (rapid disable/enable
		// thrash on the same scope), drop this enqueue. Different fleet_ids
		// or different datasets compare unequal under JSON value equality and
		// are kept. A pre-existing race window where two enqueues both see
		// "no pending" produces at most one redundant job; the handler is
		// near-idempotent so the cost is one extra walk.
		exists, err := enq.HasQueuedJobWithArgs(ctx, jobName, raw)
		if err != nil {
			return fmt.Errorf("check pending %s for %s: %w", jobName, c.scrubDataset, err)
		}
		if exists {
			continue
		}

		job := &Job{
			Name:  jobName,
			Args:  &raw,
			State: JobStateQueued,
		}
		if _, err := enq.NewJob(ctx, job); err != nil {
			return fmt.Errorf("enqueue %s for %s: %w", jobName, c.scrubDataset, err)
		}
	}
	return nil
}
