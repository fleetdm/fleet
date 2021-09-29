package async

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func BenchmarkLabelMembershipInsert(b *testing.B) {
	ds := mysql.CreateMySQLDS(b)

	const targetRows = 100_000
	batchSizes := []int{10, 100, 1_000, 2_000, 5_000, 10_000}

	b.Run("InsertOnly", func(b *testing.B) {
		var labelID uint
		for _, bsize := range batchSizes {
			b.Run(fmt.Sprint(bsize), func(b *testing.B) {
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
			})
		}
	})

	b.Run("UpdateOnly", func(b *testing.B) {
		for _, bsize := range batchSizes {
			b.Run(fmt.Sprint(bsize), func(b *testing.B) {
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
			})
		}
	})
}

func insertLabelMembershipBatch(b *testing.B, ds fleet.Datastore, batch [][2]uint) {
	ctx := context.Background()

	sql := `INSERT INTO label_membership (label_id, host_id) VALUES `
	sql += strings.Repeat(`(?, ?),`, len(batch))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	err := ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		if err != nil {
			b.Logf("exec context failed (might retry): %v", err)
		}
		return err
	})
	if err != nil {
		b.Fatal(err)
	}
}
