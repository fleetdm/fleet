package async

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/jmoiron/sqlx"
)

// On a dev laptop, I get those results to upsert 100K rows. It seems like a
// batch size of 2000-5000 range is optimal when considering a mix of inserts
// and updates. For deletes, 1000-2000 seems optimal, 10_000 crashes mysql
// due to the thread stack size limit (this is also for 100K deletions).
//
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/server/service/async
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                                           iter        time/iter
// ---------                                           ----        ---------
// BenchmarkLabelMembershipInsert/InsertOnly/10-8         1   21201.01 ms/op
// BenchmarkLabelMembershipInsert/InsertOnly/100-8        1    1299.94 ms/op
// BenchmarkLabelMembershipInsert/InsertOnly/1000-8       2     519.41 ms/op
// BenchmarkLabelMembershipInsert/InsertOnly/2000-8       2     501.45 ms/op
// BenchmarkLabelMembershipInsert/InsertOnly/5000-8       2     575.95 ms/op
// BenchmarkLabelMembershipInsert/InsertOnly/10000-8      2     759.41 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/10-8         1    9170.87 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/100-8        1    1512.05 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/1000-8       2     730.94 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/2000-8       2     588.03 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/5000-8       2     529.61 ms/op
// BenchmarkLabelMembershipInsert/UpdateOnly/10000-8      2     609.06 ms/op
// ok      github.com/fleetdm/fleet/v4/server/service/async        48.363s
//
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/server/service/async
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// PASS
// benchmark                               iter        time/iter
// ---------                               ----        ---------
// BenchmarkLabelMembershipDelete/10-8        1   10905.79 ms/op
// BenchmarkLabelMembershipDelete/100-8       1    2528.57 ms/op
// BenchmarkLabelMembershipDelete/1000-8      1    1715.99 ms/op
// BenchmarkLabelMembershipDelete/2000-8      1    1410.87 ms/op
// BenchmarkLabelMembershipDelete/3000-8      1    1653.89 ms/op
// ok      github.com/fleetdm/fleet/v4/server/service/async        24.291s

func BenchmarkLabelMembershipInsert(b *testing.B) {
	ds := mysql.CreateMySQLDS(b)

	const targetRows = 100_000
	batchSizes := []int{10, 100, 1_000, 2_000, 5_000, 10_000}

	b.Run("InsertOnly", func(b *testing.B) {
		var labelID uint
		for _, bsize := range batchSizes {
			b.Run(fmt.Sprint(bsize), func(b *testing.B) {
				defer mysql.TruncateTables(b, ds)

				batch := make([][2]uint, bsize)
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					var count int
					for {
						for i := range batch {
							labelID++
							batch[i] = [2]uint{labelID, 1}
						}
						count += len(batch)
						insertLabelMembershipBatch(b, ds, batch)
						if count >= targetRows {
							break
						}
					}
				}

				// sanity check
				mysql.ExecAdhocSQL(b, ds, func(tx sqlx.ExtContext) error {
					var count int
					if err := sqlx.GetContext(context.Background(), tx, &count, `SELECT COUNT(*) FROM label_membership`); err != nil {
						b.Logf("select count sanity check failed: %v", err)
					}
					b.Logf("count after run: %d", count)
					return nil
				})
			})
		}
	})

	b.Run("UpdateOnly", func(b *testing.B) {
		for _, bsize := range batchSizes {
			b.Run(fmt.Sprint(bsize), func(b *testing.B) {
				defer mysql.TruncateTables(b, ds)

				// insert the batch before the benchmark, then always process the
				// same batch
				var labelID uint
				batch := make([][2]uint, bsize)
				for i := range batch {
					labelID++
					batch[i] = [2]uint{labelID, 1}
				}
				insertLabelMembershipBatch(b, ds, batch)
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					var count int
					for {
						count += len(batch)
						insertLabelMembershipBatch(b, ds, batch)
						if count >= targetRows {
							break
						}
					}
				}

				// sanity check
				mysql.ExecAdhocSQL(b, ds, func(tx sqlx.ExtContext) error {
					var count int
					if err := sqlx.GetContext(context.Background(), tx, &count, `SELECT COUNT(*) FROM label_membership`); err != nil {
						b.Logf("select count sanity check failed: %v", err)
					}
					b.Logf("count after run: %d", count)
					return nil
				})
			})
		}
	})
}

func BenchmarkLabelMembershipDelete(b *testing.B) {
	ds := mysql.CreateMySQLDS(b)

	const (
		initialRows = 1_000_000
		targetRows  = 100_000
	)
	batchSizes := []int{10, 100, 1_000, 2_000, 3_000} // 10K is too big, thread stack overrun in mysql

	// insert initialRows before all benchmarks, should be enough
	var count int
	var labelID uint
	insBatch := make([][2]uint, 5000)
	for count < initialRows {
		for i := range insBatch {
			labelID++
			insBatch[i] = [2]uint{labelID, 1}
		}
		insertLabelMembershipBatch(b, ds, insBatch)
		count += len(insBatch)
	}

	for _, bsize := range batchSizes {
		b.Run(fmt.Sprint(bsize), func(b *testing.B) {
			batch := make([][2]uint, bsize)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var count int
				for {
					for i := range batch {
						labelID--
						if labelID == 0 {
							b.Fatal("ran out of rows to delete")
						}
						batch[i] = [2]uint{labelID, 1}
					}
					count += len(batch)
					deleteLabelMembershipBatch(b, ds, batch)
					if count >= targetRows {
						break
					}
				}
			}

			// sanity check
			mysql.ExecAdhocSQL(b, ds, func(tx sqlx.ExtContext) error {
				var count int
				if err := sqlx.GetContext(context.Background(), tx, &count, `SELECT COUNT(*) FROM label_membership`); err != nil {
					b.Logf("select count sanity check failed: %v", err)
				}
				b.Logf("count after run: %d", count)
				return nil
			})
		})
	}
}

func deleteLabelMembershipBatch(b *testing.B, ds *mysql.Datastore, batch [][2]uint) {
	ctx := context.Background()

	rest := strings.Repeat(`UNION ALL SELECT ?, ? `, len(batch)-1)
	sql := fmt.Sprintf(`
    DELETE
      lm
    FROM
      label_membership lm
    JOIN
      (SELECT ? label_id, ? host_id %s) del_list
    ON
      lm.label_id = del_list.label_id AND
      lm.host_id = del_list.host_id`, rest)

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	mysql.ExecAdhocSQL(b, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		if err != nil {
			b.Logf("exec context failed (might retry): %v", err)
		}
		return err
	})
}

func insertLabelMembershipBatch(b *testing.B, ds *mysql.Datastore, batch [][2]uint) {
	ctx := context.Background()

	sql := `INSERT INTO label_membership (label_id, host_id) VALUES `
	sql += strings.Repeat(`(?, ?),`, len(batch))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	mysql.ExecAdhocSQL(b, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		if err != nil {
			b.Logf("exec context failed (might retry): %v", err)
		}
		return err
	})
}
