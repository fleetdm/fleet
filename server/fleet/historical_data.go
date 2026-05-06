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
//
// HasQueuedJobWithArgs gates NewJob so rapid disable/enable toggles of the
// same scope don't stack identical scrub jobs in the queue. See design
// decision 5 of the chart-disabling-collection-scrub change.
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
// Per-iteration errors (activity emit OR scrub enqueue) are collected and
// joined rather than returned eagerly. Activity errors and scrub-enqueue
// errors are both non-fatal: a single failed audit-log emission must not
// abandon a sibling sub-key's scrub, and vice versa. Callers
// (ModifyAppConfig, ModifyTeam, editTeamFromSpec) log-and-continue on the
// returned error. See design decision 8a.
//
// Scrub-first ordering on disable flips is deliberate: an activity emit
// failure must not skip the scrub. The scrub guarantee ("disabled data
// eventually goes away") outranks the audit-log entry, and once the config
// commit lands, a retry of the API call sees `oldHD == newHD` and short-
// circuits — so any side effect dropped here is dropped permanently.
//
// fleetID and fleetName are nil for global toggles and populated for
// per-fleet toggles. Activity payload uses the public config sub-key
// ("uptime", "vulnerabilities"). Scrub job payload uses the internal dataset
// name ("uptime", "cve") — see design decision 5.
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
//
// Dedup: if an identical job is already queued (rapid disable/enable thrash
// on the same scope), the new enqueue is dropped. Different fleet_ids or
// different datasets compare unequal under JSON value equality and are kept.
// A pre-existing race window where two enqueues both see "no pending"
// produces at most one redundant job; the handler is near-idempotent so the
// cost is one extra walk.
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
