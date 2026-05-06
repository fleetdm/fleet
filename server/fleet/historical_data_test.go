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
// OnHistoricalDataChanged invocations within a test to simulate "the queue
// persists between API calls."
type fakeJobEnqueuer struct {
	jobs       []*Job
	err        error
	hasJobErr  error
	hasJobHits int
	// hasJobErrForDataset, if non-empty, makes HasQueuedJobWithArgs return
	// (false, hasJobErr) only when the args payload contains this dataset
	// string. Used to simulate per-dataset transient failures so the
	// loop-continue behavior in OnHistoricalDataChanged is exercised.
	hasJobErrForDataset string
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
		if f.hasJobErrForDataset == "" || bytes.Contains(args, []byte(`"dataset":"`+f.hasJobErrForDataset+`"`)) {
			return false, f.hasJobErr
		}
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

// fakeActivityEmitter captures NewActivity calls, with optional fixed-error
// injection per call.
type fakeActivityEmitter struct {
	activities []ActivityDetails
	err        error
}

func (f *fakeActivityEmitter) NewActivity(_ context.Context, _ *User, a ActivityDetails) error {
	if f.err != nil {
		return f.err
	}
	f.activities = append(f.activities, a)
	return nil
}

func TestOnHistoricalDataChanged_Global(t *testing.T) {
	t.Run("flips false→false: no enqueue, no activity", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			nil, nil,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
		assert.Empty(t, emit.activities)
	})

	t.Run("flips false→true: emits enabled activity, no enqueue", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			nil, nil,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
		require.Len(t, emit.activities, 2)
		for _, a := range emit.activities {
			_, ok := a.(ActivityTypeEnabledHistoricalDataset)
			assert.True(t, ok, "expected enabled activity, got %T", a)
		}
	})

	t.Run("flips true→false on vulnerabilities: enqueues 'cve' scrub and emits 'vulnerabilities' activity", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: false},
			nil, nil,
		)
		require.NoError(t, err)

		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		var args chartScrubGlobalArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		// Storage uses the internal dataset name; "vulnerabilities" would
		// match no rows in host_scd_data.
		assert.Equal(t, "cve", args.Dataset)

		require.Len(t, emit.activities, 1)
		dis, ok := emit.activities[0].(ActivityTypeDisabledHistoricalDataset)
		require.True(t, ok)
		// Activity payload uses the public sub-key.
		assert.Equal(t, "vulnerabilities", dis.Dataset)
	})

	t.Run("flips true→false on uptime: enqueues uptime scrub and emits uptime activity", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			nil, nil,
		)
		require.NoError(t, err)

		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		var args chartScrubGlobalArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		assert.Equal(t, "uptime", args.Dataset)

		require.Len(t, emit.activities, 1)
		dis, ok := emit.activities[0].(ActivityTypeDisabledHistoricalDataset)
		require.True(t, ok)
		assert.Equal(t, "uptime", dis.Dataset)
	})

	t.Run("flips true→false on both: two scrubs, two activities", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			nil, nil,
		)
		require.NoError(t, err)

		require.Len(t, enq.jobs, 2)
		datasets := []string{}
		for _, j := range enq.jobs {
			var a chartScrubGlobalArgs
			require.NoError(t, json.Unmarshal(*j.Args, &a))
			datasets = append(datasets, a.Dataset)
		}
		// Internal dataset names — see OnHistoricalDataChanged.
		assert.ElementsMatch(t, []string{"uptime", "cve"}, datasets)

		require.Len(t, emit.activities, 2)
	})
}

func TestOnHistoricalDataChanged_Fleet(t *testing.T) {
	t.Run("flips true→false on uptime: fleet-scoped scrub with this team's id", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		teamID := uint(7)
		teamName := "Test Team"
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			&teamID, &teamName,
		)
		require.NoError(t, err)

		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_fleet", enq.jobs[0].Name)
		var args chartScrubFleetArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		assert.Equal(t, "uptime", args.Dataset)
		assert.Equal(t, []uint{7}, args.FleetIDs)

		require.Len(t, emit.activities, 1)
	})

	t.Run("no flip: no enqueue, no activity", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		teamID := uint(7)
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			&teamID, nil,
		)
		require.NoError(t, err)
		assert.Empty(t, enq.jobs)
		assert.Empty(t, emit.activities)
	})

	t.Run("propagates enqueue errors", func(t *testing.T) {
		boom := errors.New("queue is closed")
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{err: boom}
		teamID := uint(7)
		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: true},
			&teamID, nil,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})
}

func TestOnHistoricalDataChanged_ScrubFirstOrdering(t *testing.T) {
	// Activity-emit failure must not skip the scrub. This is the bug the
	// merged helper exists to prevent: pre-merge, the activity error was
	// fatal and returned before the enqueue, so on retry oldHD == newHD
	// short-circuited and the scrub was permanently dropped.
	emit := &fakeActivityEmitter{err: errors.New("audit log down")}
	enq := &fakeJobEnqueuer{}

	err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
		HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
		HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
		nil, nil,
	)
	require.Error(t, err, "activity errors propagate via errors.Join")

	// Both scrubs still enqueued.
	require.Len(t, enq.jobs, 2, "scrub must enqueue even when activity emit fails")
}

func TestOnHistoricalDataChanged_Dedup(t *testing.T) {
	t.Run("rapid disable→enable→disable on global cve produces one job", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: true, Vulnerabilities: false}

		// First disable enqueues.
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil))
		// Re-enable does nothing on the scrub side.
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, off, on, nil, nil))
		// Second disable observes the still-pending job and dedups.
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil))
		// Third disable also dedups.
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil))

		require.Len(t, enq.jobs, 1)
		assert.Equal(t, "chart_scrub_dataset_global", enq.jobs[0].Name)
		assert.GreaterOrEqual(t, enq.hasJobHits, 1)
	})

	t.Run("different datasets do not dedup", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		offUptime := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}
		offCVE := HistoricalDataSettings{Uptime: true, Vulnerabilities: false}

		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, offUptime, nil, nil))
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, offCVE, nil, nil))

		require.Len(t, enq.jobs, 2)
	})

	t.Run("different fleet_ids do not dedup", func(t *testing.T) {
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}
		team5, team7 := uint(5), uint(7)

		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, &team5, nil))
		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, &team7, nil))

		require.Len(t, enq.jobs, 2)
	})

	t.Run("non-queued state does not block new enqueue", func(t *testing.T) {
		// Simulate the existing job having already been picked up: state
		// changed to something other than queued. New disable should still
		// enqueue.
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}

		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil))
		require.Len(t, enq.jobs, 1)
		// Worker started running it.
		enq.jobs[0].State = JobStateSuccess

		require.NoError(t, OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil))
		require.Len(t, enq.jobs, 2, "completed job should not block a fresh enqueue")
	})

	t.Run("propagates HasQueuedJobWithArgs errors", func(t *testing.T) {
		boom := errors.New("db down")
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{hasJobErr: boom}
		on := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		off := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}

		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil, on, off, nil, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("one dataset failing does not abandon the others", func(t *testing.T) {
		// Transient DB hiccup on the uptime iteration must not skip the cve
		// enqueue: callers (ModifyAppConfig etc.) log-and-continue on this
		// function's error precisely because once SaveAppConfig has committed,
		// "no scrub for any dataset" is strictly worse than "scrub for the
		// datasets we could".
		boom := errors.New("transient db hiccup on uptime")
		emit := &fakeActivityEmitter{}
		enq := &fakeJobEnqueuer{hasJobErr: boom, hasJobErrForDataset: "uptime"}

		err := OnHistoricalDataChanged(t.Context(), emit, enq, nil,
			HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
			HistoricalDataSettings{Uptime: false, Vulnerabilities: false},
			nil, nil,
		)
		require.Error(t, err)
		require.ErrorIs(t, err, boom, "uptime error must surface")

		require.Len(t, enq.jobs, 1, "cve scrub must still be enqueued despite uptime failure")
		var args chartScrubGlobalArgs
		require.NoError(t, json.Unmarshal(*enq.jobs[0].Args, &args))
		assert.Equal(t, "cve", args.Dataset)
	})
}
