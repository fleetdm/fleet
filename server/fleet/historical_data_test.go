package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJobEnqueuer captures NewJob calls.
type fakeJobEnqueuer struct {
	jobs []*Job
	err  error
}

func (f *fakeJobEnqueuer) NewJob(_ context.Context, j *Job) (*Job, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.jobs = append(f.jobs, j)
	return j, nil
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
