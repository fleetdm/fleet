package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestInsertUpdateCronStats(t *testing.T) {
	const (
		scheduleName = "test_sched"
		instanceID   = "test_instance"
	)
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	id, err := ds.InsertCronStats(ctx, fleet.CronStatsTypeScheduled, scheduleName, instanceID, fleet.CronStatsStatusPending)
	require.NoError(t, err)

	res, err := ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, id, res[0].ID)
	require.Equal(t, fleet.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusPending, res[0].Status)

	err = ds.UpdateCronStats(ctx, id, fleet.CronStatsStatusCompleted, &fleet.CronScheduleErrors{
		"some_job":       errors.New("some error"),
		"some_other_job": errors.New("some other error"),
	})
	require.NoError(t, err)

	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, id, res[0].ID)
	require.Equal(t, fleet.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusCompleted, res[0].Status)

	// Make sure we got valid JSON back.
	var actualMap map[string]string
	err = json.Unmarshal([]byte(res[0].Errors), &actualMap)
	require.NoError(t, err)

	// Compare the error JSON with the expected object.
	expectedJSON := `{"some_job": "some error", "some_other_job": "some other error"}`
	var expectedMap map[string]string
	err = json.Unmarshal([]byte(expectedJSON), &expectedMap)
	require.NoError(t, err)
	require.Equal(t, actualMap, expectedMap)
}

func TestGetLatestCronStats(t *testing.T) {
	const (
		scheduleName = "test_sched"
		instanceID   = "test_instance"
	)
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	insertTestCS := func(name string, statsType fleet.CronStatsType, status fleet.CronStatsStatus, createdAt time.Time) {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status, created_at) VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, statsType, name, instanceID, status, createdAt)
		require.NoError(t, err)
	}

	then := time.Now().UTC().Truncate(time.Second).Add(-24 * time.Hour)

	// insert two "scheduled" stats
	insertTestCS(scheduleName, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusPending, then.Add(2*time.Minute))
	insertTestCS(scheduleName, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusCompleted, then.Add(1*time.Minute))

	// most recent record is returned for "scheduled" stats type
	res, err := ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, fleet.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)

	// insert two "triggered" stats
	insertTestCS(scheduleName, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCompleted, then.Add(2*time.Hour))
	insertTestCS(scheduleName, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCompleted, then.Add(1*time.Hour))

	// most recent record is returned for both "scheduled" stats type and "triggered" stats type
	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, fleet.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)
	require.Equal(t, fleet.CronStatsTypeTriggered, res[1].StatsType)
	require.Equal(t, fleet.CronStatsStatusCompleted, res[1].Status)
	require.Equal(t, then.Add(2*time.Hour), res[1].CreatedAt)

	// insert some other stats that shouldn't be returned
	insertTestCS(scheduleName, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusExpired, then.Add(3*time.Hour))    // expired status shouldn't be returned
	insertTestCS(scheduleName, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusExpired, then.Add(3*time.Hour))    // expired status shouldn't be returned
	insertTestCS(scheduleName, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusCanceled, then.Add(4*time.Hour))   // canceled status shouldn't be returned
	insertTestCS(scheduleName, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCanceled, then.Add(4*time.Hour))   // canceled status shouldn't be returned
	insertTestCS("schedule_1337", fleet.CronStatsTypeTriggered, fleet.CronStatsStatusPending, then.Add(5*time.Hour)) // different name shouldn't be returned

	// most recent record is returned for both "scheduled" stats type and "triggered" stats type
	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, fleet.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)
	require.Equal(t, fleet.CronStatsTypeTriggered, res[1].StatsType)
	require.Equal(t, fleet.CronStatsStatusCompleted, res[1].Status)
	require.Equal(t, then.Add(2*time.Hour), res[1].CreatedAt)
}

func TestCleanupCronStats(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)
	now := time.Now().UTC().Truncate(time.Second)
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
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
			createdAt:               twoDaysAgo.Add(1 * time.Hour),
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               twoDaysAgo.Add(-1 * time.Hour),
			status:                  fleet.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     true,
		},
	}

	for _, c := range cases {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status, created_at) VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, fleet.CronStatsTypeScheduled, name, instance, c.status, c.createdAt)
		require.NoError(t, err)
	}

	var stats []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, s := range stats {
		require.Equal(t, cases[i].createdAt, s.CreatedAt)
		require.Equal(t, cases[i].status, s.Status)
	}

	err = ds.CleanupCronStats(ctx)
	require.NoError(t, err)

	stats = []fleet.CronStats{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
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
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, fleet.CronStatsTypeScheduled, c.schedName, c.instance, c.status)
		require.NoError(t, err)
	}

	var stats []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
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
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
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
