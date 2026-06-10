package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// HistoricalDataActivityEmitter is the activity-emit dependency of
// OnHistoricalDataChanged. Both the free service and EE service satisfy it
// via NewActivity.
type HistoricalDataActivityEmitter interface {
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
}

// HistoricalDataScrubEnqueuer is the scrub-enqueue dependency of
// OnHistoricalDataChanged. fleet.Datastore satisfies it via NewJob and
// HasQueuedJobWithArgs. Defined locally to avoid pulling worker-package
// types into the fleet package.
type HistoricalDataScrubEnqueuer interface {
	NewJob(ctx context.Context, j *Job) (*Job, error)
	HasQueuedJobWithArgs(ctx context.Context, name string, args json.RawMessage) (bool, error)
}

// OnHistoricalDataChanged is the hook called when historical_data config
// changes. For each historical_data sub-key whose value differs between oldHD
// and newHD, it:
//
//   - on a disable flip (true→false): enqueues a scrub job, then emits the
//     "disabled" activity.
//   - on an enable flip (false→true): emits the "enabled" activity. No scrub.
//
// CALLER ORDERING REQUIREMENT: this function MUST be called AFTER the
// corresponding SaveAppConfig / SaveTeam commit has succeeded. Enqueuing
// before commit risks a worker picking up the job before the new config is
// visible, which would race the collection cron and re-introduce the bits
// the scrub is meant to clear.
func OnHistoricalDataChanged(
	ctx context.Context,
	emitter HistoricalDataActivityEmitter,
	enq HistoricalDataScrubEnqueuer,
	user *User,
	oldHD, newHD HistoricalDataSettings,
	fleetID *uint, fleetName *string,
) error {
	changes := []struct {
		// configKey is the public sub-key, used in activity payloads.
		configKey string
		// scrubDataset is the internal dataset name (matches host_scd_data.dataset
		// and chart-package Dataset.Name() return values), used in scrub job payloads.
		scrubDataset string
		oldVal       bool
		newVal       bool
	}{
		{"uptime", "uptime", oldHD.Uptime, newHD.Uptime},
		{"vulnerabilities", "cve", oldHD.Vulnerabilities, newHD.Vulnerabilities},
	}
	var errs []error
	for _, c := range changes {
		if c.oldVal == c.newVal {
			continue
		}
		// Disable flip: enqueue scrub before emitting the activity so an
		// activity-emit failure can't drop the scrub silently.
		if !c.newVal {
			if err := enqueueHistoricalDataScrub(ctx, enq, c.scrubDataset, fleetID); err != nil {
				errs = append(errs, fmt.Errorf("enqueue scrub for %s: %w", c.scrubDataset, err))
			}
		}
		var act ActivityDetails
		if c.newVal {
			act = ActivityTypeEnabledHistoricalDataset{
				Dataset:   c.configKey,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		} else {
			act = ActivityTypeDisabledHistoricalDataset{
				Dataset:   c.configKey,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		}
		if err := emitter.NewActivity(ctx, user, act); err != nil {
			errs = append(errs, fmt.Errorf("emit activity for %s: %w", c.configKey, err))
		}
	}
	return errors.Join(errs...)
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

// enqueueHistoricalDataScrub inserts a scrub job for one (dataset, scope) flip.
//
// fleetID semantics:
//   - nil: a global flip. Enqueues one chart_scrub_dataset_global job that
//     will DELETE every row for the dataset.
//   - non-nil: a per-fleet flip. Enqueues one chart_scrub_dataset_fleet job
//     whose fleet_ids slice contains just this fleet ID. (Per-call
//     coalescing across multiple teams within a single batch is a future
//     optimization — see the chart-historical-data-collection spec.)
func enqueueHistoricalDataScrub(
	ctx context.Context,
	enq HistoricalDataScrubEnqueuer,
	scrubDataset string,
	fleetID *uint,
) error {
	var jobName string
	var argsJSON []byte
	var err error
	if fleetID == nil {
		jobName = chartScrubDatasetGlobalJobName
		argsJSON, err = json.Marshal(chartScrubGlobalArgs{Dataset: scrubDataset})
	} else {
		jobName = chartScrubDatasetFleetJobName
		argsJSON, err = json.Marshal(chartScrubFleetArgs{
			Dataset:  scrubDataset,
			FleetIDs: []uint{*fleetID},
		})
	}
	if err != nil {
		return fmt.Errorf("marshal scrub job args: %w", err)
	}
	raw := json.RawMessage(argsJSON)

	// Check for an existing queued job with the same name and args to avoid
	// redundant scrubs on rapid disable→enable→disable thrash.
	exists, err := enq.HasQueuedJobWithArgs(ctx, jobName, raw)
	if err != nil {
		return fmt.Errorf("check pending %s: %w", jobName, err)
	}
	if exists {
		return nil
	}

	job := &Job{
		Name:  jobName,
		Args:  &raw,
		State: JobStateQueued,
	}
	if _, err := enq.NewJob(ctx, job); err != nil {
		return fmt.Errorf("enqueue %s: %w", jobName, err)
	}
	return nil
}
