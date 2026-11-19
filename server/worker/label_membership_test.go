package worker

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelMembershipCleanup(t *testing.T) {
	ctx := t.Context()

	ds := mysqltest.CreateMySQLDS(t)
	// A DB migration may have created jobs.
	mysqltest.TruncateTables(t, ds)

	w := NewWorker(ds, slog.New(slog.DiscardHandler))
	w.Register(&LabelMembershipCleanup{Datastore: ds})

	var hosts []*fleet.Host
	for i := range 3 {
		name := fmt.Sprintf("label-membership-cleanup-%d", i)
		hosts = append(hosts, test.NewHost(t, ds, name, "", name, name, time.Now()))
	}
	newLabelWithAllHosts := func(name string, teamID *uint) *fleet.Label {
		label, err := ds.NewLabel(ctx, &fleet.Label{
			Name:                name,
			Query:               "SELECT 1",
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
			TeamID:              teamID,
		})
		require.NoError(t, err)
		for _, h := range hosts {
			require.NoError(t, ds.RecordLabelQueryExecutions(ctx, h, map[uint]*bool{label.ID: new(true)}, time.Now(), false))
		}
		return label
	}
	membershipCount := func(labelID uint) int {
		hostIDs, err := ds.LabelMembershipHostIDs(ctx, labelID)
		require.NoError(t, err)
		return len(hostIDs)
	}
	requireNoQueuedJobs := func() {
		jobs, err := ds.GetQueuedJobs(ctx, 10, time.Time{})
		require.NoError(t, err)
		require.Empty(t, jobs)
	}

	kept := newLabelWithAllHosts("label-membership-cleanup-kept", nil)

	t.Run("deleted label", func(t *testing.T) {
		deleted := newLabelWithAllHosts("label-membership-cleanup-deleted", nil)

		require.NoError(t, ds.DeleteLabel(ctx, deleted.ID))
		require.Equal(t, len(hosts), membershipCount(deleted.ID))

		require.NoError(t, QueueLabelMembershipCleanupJob(ctx, ds))
		require.NoError(t, w.ProcessJobs(ctx))

		assert.Zero(t, membershipCount(deleted.ID))
		assert.Equal(t, len(hosts), membershipCount(kept.ID))
		requireNoQueuedJobs()
	})

	t.Run("deleted fleet", func(t *testing.T) {
		fleetWithLabels, err := ds.NewTeam(ctx, &fleet.Team{Name: "label-membership-cleanup-fleet"})
		require.NoError(t, err)
		firstLabel := newLabelWithAllHosts("label-membership-cleanup-fleet-first", &fleetWithLabels.ID)
		secondLabel := newLabelWithAllHosts("label-membership-cleanup-fleet-second", &fleetWithLabels.ID)

		require.NoError(t, ds.DeleteTeam(ctx, fleetWithLabels.ID))
		require.Equal(t, len(hosts), membershipCount(firstLabel.ID))
		require.Equal(t, len(hosts), membershipCount(secondLabel.ID))

		require.NoError(t, QueueLabelMembershipCleanupJob(ctx, ds))
		require.NoError(t, w.ProcessJobs(ctx))

		assert.Zero(t, membershipCount(firstLabel.ID))
		assert.Zero(t, membershipCount(secondLabel.ID))
		assert.Equal(t, len(hosts), membershipCount(kept.ID))
		requireNoQueuedJobs()
	})
}
