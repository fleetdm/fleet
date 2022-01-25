package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/stretchr/testify/require"
)

func BenchmarkCalculateHostsPerSoftware(b *testing.B) {
	ts := time.Now()
	type counts struct{ hs, sws int }

	cases := []counts{
		{1, 1},
		{10, 10},
		{100, 100},
		{1000, 100},
		{10000, 100},
		{10000, 1000},
	}

	/*
		b.Run("resetTruncate", func(b *testing.B) {
			b.Run("singleSelectGroupByInsertOneAtTime", func(b *testing.B) {
				for _, c := range cases {
					b.Run(fmt.Sprintf("%d:%d", c.hs, c.sws), func(b *testing.B) {
						ds := CreateMySQLDS(b)
						generateHostsWithSoftware(b, ds, c.hs, c.sws)
						b.ResetTimer()

						for i := 0; i < b.N; i++ {
							resetTruncateTable(b, ds)
							singleSelectGroupByInsertOneAtTime(b, ds, ts)
						}
						checkCounts(b, ds, c.hs, c.sws)
					})
				}
			})

			b.Run("singleSelectGroupByInsertBatch100", func(b *testing.B) {
				for _, c := range cases {
					b.Run(fmt.Sprintf("%d:%d", c.hs, c.sws), func(b *testing.B) {
						ds := CreateMySQLDS(b)
						generateHostsWithSoftware(b, ds, c.hs, c.sws)
						b.ResetTimer()

						for i := 0; i < b.N; i++ {
							resetTruncateTable(b, ds)
							singleSelectGroupByInsertBatch(b, ds, ts, 100)
						}
						checkCounts(b, ds, c.hs, c.sws)
					})
				}
			})
		})
	*/

	b.Run("resetUpdate", func(b *testing.B) {
		//b.Run("singleSelectGroupByInsertOneAtTime", func(b *testing.B) {
		//	for _, c := range cases {
		//		b.Run(fmt.Sprintf("%d:%d", c.hs, c.sws), func(b *testing.B) {
		//			ds := CreateMySQLDS(b)
		//			generateHostsWithSoftware(b, ds, c.hs, c.sws)
		//			b.ResetTimer()

		//			for i := 0; i < b.N; i++ {
		//				resetUpdateAllZero(b, ds)
		//				singleSelectGroupByInsertOneAtTime(b, ds, ts)
		//			}
		//			checkCounts(b, ds, c.hs, c.sws)
		//		})
		//	}
		//})

		//b.Run("singleSelectGroupByInsertBatch100", func(b *testing.B) {
		//	for _, c := range cases {
		//		b.Run(fmt.Sprintf("%d:%d", c.hs, c.sws), func(b *testing.B) {
		//			ds := CreateMySQLDS(b)
		//			generateHostsWithSoftware(b, ds, c.hs, c.sws)
		//			b.ResetTimer()

		//			for i := 0; i < b.N; i++ {
		//				resetUpdateAllZero(b, ds)
		//				singleSelectGroupByInsertBatch(b, ds, ts, 100)
		//			}
		//			checkCounts(b, ds, c.hs, c.sws)
		//		})
		//	}
		//})

		b.Run("singleSelectGroupByInsertBatch100AggStats", func(b *testing.B) {
			for _, c := range cases {
				b.Run(fmt.Sprintf("%d:%d", c.hs, c.sws), func(b *testing.B) {
					ds := CreateMySQLDS(b)
					generateHostsWithSoftware(b, ds, c.hs, c.sws)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						resetUpdateAllZeroAgg(b, ds)
						singleSelectGroupByInsertBatchAgg(b, ds, ts, 100)
					}
					checkCountsAgg(b, ds, c.hs, c.sws)
				})
			}
		})
	})
}

func checkCounts(b *testing.B, ds *Datastore, hs, sws int) {
	var rowsCount, invalidHostsCount int

	rowsStmt := `SELECT COUNT(*) FROM software_host_counts`
	err := ds.writer.GetContext(context.Background(), &rowsCount, rowsStmt)
	require.NoError(b, err)
	require.Equal(b, sws, rowsCount)

	invalidStmt := `SELECT COUNT(*) FROM software_host_counts WHERE hosts_count != ?`
	err = ds.writer.GetContext(context.Background(), &invalidHostsCount, invalidStmt, hs)
	require.NoError(b, err)
	require.Equal(b, 0, invalidHostsCount)
}

func checkCountsAgg(b *testing.B, ds *Datastore, hs, sws int) {
	var rowsCount, invalidHostsCount int

	rowsStmt := `SELECT COUNT(*) FROM aggregated_stats WHERE type = "software_hosts_count"`
	err := ds.writer.GetContext(context.Background(), &rowsCount, rowsStmt)
	require.NoError(b, err)
	require.Equal(b, sws, rowsCount)

	invalidStmt := `SELECT COUNT(*) FROM aggregated_stats WHERE type = "software_hosts_count" AND json_value != CAST(? AS json)`
	err = ds.writer.GetContext(context.Background(), &invalidHostsCount, invalidStmt, hs)
	require.NoError(b, err)
	require.Equal(b, 0, invalidHostsCount)
}

func generateHostsWithSoftware(b *testing.B, ds *Datastore, hs, sws int) {
	hostInsert := `
	INSERT INTO hosts (
		osquery_host_id,
		node_key,
		hostname,
		uuid
	)
	VALUES `
	hostValuePart := `(?, ?, ?, ?),`

	var sb strings.Builder
	sb.WriteString(hostInsert)
	args := make([]interface{}, 0, hs*4)
	for i := 0; i < hs; i++ {
		osqueryHostID, _ := server.GenerateRandomText(10)
		name := "host" + strconv.Itoa(i)
		args = append(args, osqueryHostID, name+"key", name, name+"uuid")
		sb.WriteString(hostValuePart)
	}
	stmt := strings.TrimSuffix(sb.String(), ",")
	_, err := ds.writer.ExecContext(context.Background(), stmt, args...)
	require.NoError(b, err)

	swInsert := `
  INSERT INTO software (
    name,
    version,
    source
  ) VALUES `
	swValuePart := `(?, ?, ?),`

	sb.Reset()
	sb.WriteString(swInsert)
	args = make([]interface{}, 0, sws*3)
	for i := 0; i < sws; i++ {
		name := "software" + strconv.Itoa(i)
		args = append(args, name, strconv.Itoa(i)+".0.0", "testing")
		sb.WriteString(swValuePart)
	}
	stmt = strings.TrimSuffix(sb.String(), ",")
	_, err = ds.writer.ExecContext(context.Background(), stmt, args...)
	require.NoError(b, err)

	// cartesian product of hosts and software tables
	hostSwInsert := `
  INSERT INTO host_software (host_id, software_id)
  SELECT
    h.id,
    sw.id
  FROM
    hosts h,
    software sw`
	_, err = ds.writer.ExecContext(context.Background(), hostSwInsert)
	require.NoError(b, err)
}

func resetTruncateTable(b *testing.B, ds *Datastore) {
	truncStmt := `TRUNCATE TABLE software_host_counts`
	_, err := ds.writer.ExecContext(context.Background(), truncStmt)
	require.NoError(b, err)
}

func resetUpdateAllZero(b *testing.B, ds *Datastore) {
	updateStmt := `UPDATE software_host_counts SET hosts_count = 0`
	_, err := ds.writer.ExecContext(context.Background(), updateStmt)
	require.NoError(b, err)
}

func resetUpdateAllZeroAgg(b *testing.B, ds *Datastore) {
	updateStmt := `UPDATE aggregated_stats SET json_value = CAST(0 AS json) WHERE type = "software_hosts_count"`
	_, err := ds.writer.ExecContext(context.Background(), updateStmt)
	require.NoError(b, err)
}

func singleSelectGroupByInsertOneAtTime(b *testing.B, ds *Datastore, updatedAt time.Time) {
	queryStmt := `
    SELECT count(*), software_id
    FROM host_software
    GROUP BY software_id`

	insertStmt := `
    INSERT INTO software_host_counts
      (software_id, hosts_count, updated_at)
    VALUES
      (?, ?, ?)
    ON DUPLICATE KEY UPDATE
      hosts_count = VALUES(hosts_count),
      updated_at = VALUES(updated_at)`

	rows, err := ds.reader.QueryContext(context.Background(), queryStmt)
	require.NoError(b, err)
	defer rows.Close()

	// use a loop to iterate to prevent loading all in one go in memory, as it
	// could get pretty big at >100K hosts with 1000+ software each.
	for rows.Next() {
		var count int
		var sid uint

		require.NoError(b, rows.Scan(&count, &sid))
		_, err := ds.writer.ExecContext(context.Background(), insertStmt, sid, count, updatedAt)
		require.NoError(b, err)
	}
	require.NoError(b, rows.Err())
}

func singleSelectGroupByInsertBatch(b *testing.B, ds *Datastore, updatedAt time.Time, batchSize int) {
	queryStmt := `
    SELECT count(*), software_id
    FROM host_software
    GROUP BY software_id`

	insertStmt := `
    INSERT INTO software_host_counts
      (software_id, hosts_count, updated_at)
    VALUES
      %s
    ON DUPLICATE KEY UPDATE
      hosts_count = VALUES(hosts_count),
      updated_at = VALUES(updated_at)`
	valuesPart := `(?, ?, ?),`

	rows, err := ds.reader.QueryContext(context.Background(), queryStmt)
	require.NoError(b, err)
	defer rows.Close()

	// use a loop to iterate to prevent loading all in one go in memory, as it
	// could get pretty big at >100K hosts with 1000+ software each.
	var batchCount int
	args := make([]interface{}, 0, batchSize*3)
	for rows.Next() {
		var count int
		var sid uint

		require.NoError(b, rows.Scan(&count, &sid))
		args = append(args, sid, count, updatedAt)
		batchCount++

		if batchCount == batchSize {
			values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
			_, err := ds.writer.ExecContext(context.Background(), fmt.Sprintf(insertStmt, values), args...)
			require.NoError(b, err)

			args = args[:0]
			batchCount = 0
		}
	}

	if batchCount > 0 {
		values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
		_, err := ds.writer.ExecContext(context.Background(), fmt.Sprintf(insertStmt, values), args...)
		require.NoError(b, err)
	}
	require.NoError(b, rows.Err())
}

func singleSelectGroupByInsertBatchAgg(b *testing.B, ds *Datastore, updatedAt time.Time, batchSize int) {
	queryStmt := `
    SELECT count(*), software_id
    FROM host_software
    GROUP BY software_id`

	insertStmt := `
    INSERT INTO aggregated_stats
      (id, type, json_value, updated_at)
    VALUES
      %s
    ON DUPLICATE KEY UPDATE
      json_value = VALUES(json_value),
      updated_at = VALUES(updated_at)`
	valuesPart := `(?, "software_hosts_count", CAST(? AS json), ?),`

	rows, err := ds.reader.QueryContext(context.Background(), queryStmt)
	require.NoError(b, err)
	defer rows.Close()

	var batchCount int
	args := make([]interface{}, 0, batchSize*3)
	for rows.Next() {
		var count int
		var sid uint

		require.NoError(b, rows.Scan(&count, &sid))
		args = append(args, sid, count, updatedAt)
		batchCount++

		if batchCount == batchSize {
			values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
			_, err := ds.writer.ExecContext(context.Background(), fmt.Sprintf(insertStmt, values), args...)
			require.NoError(b, err)

			args = args[:0]
			batchCount = 0
		}
	}

	if batchCount > 0 {
		values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
		_, err := ds.writer.ExecContext(context.Background(), fmt.Sprintf(insertStmt, values), args...)
		require.NoError(b, err)
	}
	require.NoError(b, rows.Err())
}
