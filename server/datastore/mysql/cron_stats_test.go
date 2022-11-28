package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetInsertUpdateCronStats(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)
	start := time.Now().UTC().Truncate(time.Second)

	cases := []fleet.CronStats{
		{
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      "sched1",
			Instance:  "inst1",
			Status:    fleet.CronStatsStatusPending,
		},
		{
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      "sched1",
			Instance:  "inst1",
			Status:    fleet.CronStatsStatusPending,
		},
		{
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      "sched2",
			Instance:  "inst1",
			Status:    fleet.CronStatsStatusPending,
		},
		{
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      "sched2",
			Instance:  "inst2",
			Status:    fleet.CronStatsStatusCompleted,
		},
		{
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      "sched2",
			Instance:  "inst2",
			Status:    fleet.CronStatsStatusPending,
		},
		{
			StatsType: fleet.CronStatsTypeTriggered,
			Name:      "sched2",
			Instance:  "inst2",
			Status:    fleet.CronStatsStatusCompleted,
		},
	}

	var results []fleet.CronStats

	for i, c := range cases {
		now := time.Now().UTC().Truncate(time.Second)
		_, err := ds.InsertCronStats(ctx, c.StatsType, c.Name, c.Instance, c.Status)
		require.NoError(t, err)
		res, err := ds.GetLatestCronStats(ctx, c.Name)
		require.NoError(t, err)
		require.Equal(t, i+1, res.ID)
		require.Equal(t, c.StatsType, res.StatsType)
		require.Equal(t, c.Name, res.Name)
		require.Equal(t, c.Instance, res.Instance)
		require.Equal(t, c.Status, res.Status)
		require.WithinRange(t, res.CreatedAt, now, now.Add(1*time.Second))
		require.WithinRange(t, res.UpdatedAt, now, now.Add(1*time.Second))
		results = append(results, res)
	}

	time.Sleep(time.Until(start.Add(2 * time.Second)))
	for _, r := range results {
		err := ds.UpdateCronStats(ctx, r.ID, fleet.CronStatsStatusCompleted)
		require.NoError(t, err)
	}

	var updatedResults []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader, &updatedResults, "SELECT * FROM cron_stats ORDER BY id")
	require.NoError(t, err)
	require.Len(t, updatedResults, len(cases))
	for i, r := range updatedResults {
		require.Equal(t, results[i].ID, r.ID)
		require.Equal(t, results[i].StatsType, r.StatsType)
		require.Equal(t, results[i].Name, r.Name)
		require.Equal(t, results[i].Instance, r.Instance)
		require.Equal(t, fleet.CronStatsStatusCompleted, r.Status)
		require.Equal(t, results[i].CreatedAt, r.CreatedAt)
		if cases[i].Status != fleet.CronStatsStatusCompleted {
			require.WithinRange(t, r.UpdatedAt, start.Add(2*time.Second), time.Now())
		} else {
			require.Equal(t, results[i].UpdatedAt, r.UpdatedAt)
		}
	}

	// GetLatestCronStats always returns the last inserted for the named schedule
	res, err := ds.GetLatestCronStats(ctx, "sched1")
	require.NoError(t, err)
	// second case was the last inserted for sched1
	require.Equal(t, 2, res.ID)
	require.Equal(t, cases[1].StatsType, res.StatsType)
	require.Equal(t, cases[1].Name, res.Name)
	require.Equal(t, cases[1].Instance, res.Instance)
	require.Equal(t, fleet.CronStatsStatusCompleted, res.Status)
	require.Equal(t, results[1].CreatedAt, res.CreatedAt)
	require.WithinRange(t, res.UpdatedAt, start.Add(2*time.Second), time.Now()) // second case was updated from pending to completed

	res, err = ds.GetLatestCronStats(ctx, "sched2")
	require.NoError(t, err)
	// sixth case was the last inserted for sched2
	require.Equal(t, 6, res.ID)
	require.Equal(t, cases[5].Name, res.Name)
	require.Equal(t, cases[5].Instance, res.Instance)
	require.Equal(t, fleet.CronStatsStatusCompleted, res.Status)
	require.Equal(t, results[5].CreatedAt, res.CreatedAt)
	require.Equal(t, results[5].CreatedAt, res.CreatedAt)
	require.Equal(t, results[5].UpdatedAt, res.UpdatedAt) // sixth case wasn't updated
}

func TestCleanupCronStats(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)
	now := time.Now().UTC().Truncate(time.Second)
	twoWeeksAgo := now.Add(-14 * 24 * time.Hour)
	name := "test_sched"
	instance := "test_instance"

	cases := []struct {
		createdAt               time.Time
		status                  fleet.CronStatsStatus
		shouldCleanupMaxPending bool
		shouldCleanupMaxAge     bool
	}{
		{
			createdAt:               now,
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now,
			status:                  fleet.CronStatsStatusPending,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-1 * time.Hour),
			status:                  fleet.CronStatsStatusPending,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-2 * time.Hour),
			status:                  fleet.CronStatsStatusExpired,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-3 * time.Hour),
			status:                  fleet.CronStatsStatusPending,
			shouldCleanupMaxPending: true,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-3 * time.Hour),
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               twoWeeksAgo.Add(1 * time.Hour),
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               twoWeeksAgo.Add(-1 * time.Hour),
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     true,
		},
	}

	for _, c := range cases {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status, created_at) VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer.ExecContext(ctx, stmt, fleet.CronStatsTypeScheduled, name, instance, c.status, c.createdAt)
		require.NoError(t, err)
	}

	var stats []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader, &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, s := range stats {
		require.Equal(t, cases[i].createdAt, s.CreatedAt)
		require.Equal(t, cases[i].status, s.Status)
	}

	err = ds.CleanupCronStats(ctx)
	require.NoError(t, err)

	stats = []fleet.CronStats{}
	err = sqlx.SelectContext(ctx, ds.reader, &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases)-1) // case[7] was deleted because it exceeded max age
	for i, c := range cases {
		if i >= len(stats) {
			require.True(t, c.shouldCleanupMaxAge)
			break
		}
		if c.shouldCleanupMaxPending {
			require.Equal(t, fleet.CronStatsStatusExpired, stats[i].Status)
		} else {
			require.Equal(t, c.status, stats[i].Status)
		}
	}
}

func TestUpdateAllCronStatsForInstance(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	cases := []struct {
		instance     string
		schedName    string
		status       fleet.CronStatsStatus
		shouldUpdate bool
	}{
		{
			instance:     "inst1",
			schedName:    "sched1",
			status:       fleet.CronStatsStatusCompleted,
			shouldUpdate: false,
		},
		{
			instance:     "inst1",
			schedName:    "sched1",
			status:       fleet.CronStatsStatusPending,
			shouldUpdate: true,
		},
		{
			instance:     "inst1",
			schedName:    "sched2",
			status:       fleet.CronStatsStatusExpired,
			shouldUpdate: false,
		},
		{
			instance:     "inst1",
			schedName:    "sched2",
			status:       fleet.CronStatsStatusPending,
			shouldUpdate: true,
		},
		{
			instance:     "inst2",
			schedName:    "sched1",
			status:       fleet.CronStatsStatusPending,
			shouldUpdate: false,
		},
		{
			instance:     "inst2",
			schedName:    "sched2",
			status:       fleet.CronStatsStatusPending,
			shouldUpdate: false,
		},
	}

	for _, c := range cases {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status) VALUES (?, ?, ?, ?)`
		_, err := ds.writer.ExecContext(ctx, stmt, fleet.CronStatsTypeScheduled, c.schedName, c.instance, c.status)
		require.NoError(t, err)
	}

	var stats []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader, &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, s := range stats {
		require.Equal(t, cases[i].schedName, s.Name)
		require.Equal(t, cases[i].instance, s.Instance)
		require.Equal(t, cases[i].status, s.Status)
	}

	err = ds.UpdateAllCronStatsForInstance(ctx, "inst1", fleet.CronStatsStatusPending, fleet.CronStatsStatusCanceled)
	require.NoError(t, err)

	stats = []fleet.CronStats{}
	err = sqlx.SelectContext(ctx, ds.reader, &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, c := range cases {
		s := stats[i]
		require.Equal(t, c.instance, s.Instance)
		require.Equal(t, c.schedName, s.Name)
		if c.shouldUpdate {
			require.Equal(t, fleet.CronStatsStatusCanceled, s.Status)
		} else {
			require.Equal(t, c.status, s.Status)
		}
	}
}
