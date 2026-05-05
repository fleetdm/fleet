package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJobEnqueuer captures NewJob calls and answers HasQueuedJobWithArgs by
// scanning the captured queue. The same fake is shared across multiple
// EnqueueHistoricalDataScrubs invocations within a test to simulate "the
// queue persists between API calls."
type fakeJobEnqueuer struct {
	jobs       []*Job
	err        error
	hasJobErr  error
	hasJobHits int
}

func (f *fakeJobEnqueuer) NewJob(_ context.Context, j *Job) (*Job, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.jobs = append(f.jobs, j)
	return j, nil
}

func (f *fakeJobEnqueuer) HasQueuedJobWithArgs(_ context.Context, name string, args json.RawMessage) (bool, error) {
	if f.hasJobErr != nil {
		return false, f.hasJobErr
	}
	for _, j := range f.jobs {
		if j.Name != name || j.State != JobStateQueued || j.Args == nil {
			continue
		}
		if bytes.Equal(*j.Args, args) {
			f.hasJobHits++
			return true, nil
		}
	}
	return false, nil
}

func TestEnqueueHistoricalDataScrubs_Global(t *testing.T) {
	t.Run("flips false→false: no enqueue", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			nil,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
	})

	t.Run("flips false→true: no enqueue", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			nil,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
	})

	t.Run("flips true→false on vulnerabilities: enqueues scrub with internal name 'cve'", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: false},
			nil,
		)
		require.NoError(t, err)
		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		var args chartScrubGlobalArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		// Storage uses the internal dataset name; "vulnerabilities" would
		// match no rows in host_scd_data.
		assert.Equal(t, "cve", args.Dataset)
	})

	t.Run("flips true→false on uptime: enqueues global uptime scrub", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			nil,
		)
		require.NoError(t, err)
		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		var args chartScrubGlobalArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		assert.Equal(t, "uptime", args.Dataset)
	})

	t.Run("flips true→false on both: enqueues two global scrubs", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			nil,
		)
		require.NoError(t, err)
		require.Len(t, enq.jobs, 2)
		datasets := []string{}
		for _, j := range enq.jobs {
			var a chartScrubGlobalArgs
			require.NoError(t, json.Unmarshal(*j.Args, &a))
			datasets = append(datasets, a.Dataset)
		}
		// Internal dataset names — see EnqueueHistoricalDataScrubs comment.
		assert.ElementsMatch(t, []string{"uptime", "cve"}, datasets)
	})
}

func TestEnqueueHistoricalDataScrubs_Fleet(t *testing.T) {
	t.Run("flips true→false on uptime: enqueues fleet scrub with this team's id", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		teamID := uint(7)
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			&teamID,
		)
		require.NoError(t, err)
		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_fleet", enq.jobs[0].Name)
		var args chartScrubFleetArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		assert.Equal(t, "uptime", args.Dataset)
		assert.Equal(t, []uint{7}, args.FleetIDs)
	})

	t.Run("no flip: no enqueue", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		teamID := uint(7)
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			&teamID,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
	})

	t.Run("propagates enqueue errors", func(t *testing.T) {
		boom := errors.New("queue is closed")
		enq := &fakeJobEnqueuer{err: boom}
		teamID := uint(7)
		err := EnqueueHistoricalDataScrubs(t.Context(), enq,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			&teamID,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})
}

func TestEnqueueHistoricalDataScrubs_Dedup(t *testing.T) {
	t.Run("rapid disable→enable→disable on global cve produces one job", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: true, Vulnerabilities: false}

		// First disable enqueues.
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil))
		// Re-enable does nothing (false→true is skipped before dedup check).
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, off, on, nil))
		// Second disable observes the still-pending job and dedups.
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil))
		// Third disable also dedups.
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil))

		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		assert.GreaterOrEqual(t, enq.hasJobHits, 1)
	})

	t.Run("different datasets do not dedup", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		offUptime := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}
		offCVE := HistoricalDataSettings{Uptime: true, Vulnerabilities: false}

		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, offUptime, nil))
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, offCVE, nil))

		require.Len(t, enq.jobs, 2)
	})

	t.Run("different fleet_ids do not dedup", func(t *testing.T) {
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}
		team5, team7 := uint(5), uint(7)

		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, &team5))
		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, &team7))

		require.Len(t, enq.jobs, 2)
	})

	t.Run("non-queued state does not block new enqueue", func(t *testing.T) {
		// Simulate the existing job having already been picked up: state
		// changed to something other than queued. New disable should still
		// enqueue.
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}

		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil))
		require.Len(t, enq.jobs, 1)
		// Worker started running it.
		enq.jobs[0].State = JobStateSuccess

		require.NoError(t, EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil))
		require.Len(t, enq.jobs, 2, "completed job should not block a fresh enqueue")
	})

	t.Run("propagates HasQueuedJobWithArgs errors", func(t *testing.T) {
		boom := errors.New("db down")
		enq := &fakeJobEnqueuer{hasJobErr: boom}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}

		err := EnqueueHistoricalDataScrubs(t.Context(), enq, on, off, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})
}
